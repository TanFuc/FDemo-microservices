package safego

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

// Logger interface for custom logging
type Logger interface {
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

// defaultLogger uses slog for logging
type defaultLogger struct{}

func (d *defaultLogger) Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func (d *defaultLogger) Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

var globalLogger Logger = &defaultLogger{}

// SetLogger sets the global logger for safe goroutines
func SetLogger(l Logger) {
	if l != nil {
		globalLogger = l
	}
}

// Go runs a function in a goroutine with panic recovery
// If the function panics, it logs the error and stack trace
func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				globalLogger.Error("goroutine panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

// GoWithContext runs a function with context in a goroutine with panic recovery
func GoWithContext(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				globalLogger.Error("goroutine panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn(ctx)
	}()
}

// GoWithCallback runs a function in a goroutine with panic recovery and error callback
func GoWithCallback(fn func() error, onError func(error), onPanic func(interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				globalLogger.Error("goroutine panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
				if onPanic != nil {
					onPanic(r)
				}
			}
		}()
		if err := fn(); err != nil {
			if onError != nil {
				onError(err)
			}
		}
	}()
}

// Pool manages a pool of goroutines with panic recovery
type Pool struct {
	wg     sync.WaitGroup
	sem    chan struct{}
	logger Logger
}

// NewPool creates a new goroutine pool with max concurrency
// If maxConcurrency <= 0, it defaults to unlimited
func NewPool(maxConcurrency int) *Pool {
	p := &Pool{
		logger: globalLogger,
	}
	if maxConcurrency > 0 {
		p.sem = make(chan struct{}, maxConcurrency)
	}
	return p
}

// Go runs a function in the pool with panic recovery
func (p *Pool) Go(fn func()) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				p.logger.Error("goroutine pool panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
			}
		}()

		if p.sem != nil {
			p.sem <- struct{}{}
			defer func() { <-p.sem }()
		}

		fn()
	}()
}

// Wait waits for all goroutines in the pool to complete
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Parallel runs multiple functions concurrently and waits for all to complete
// Returns errors from any functions that failed
func Parallel(fns ...func() error) []error {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		errors []error
	)

	wg.Add(len(fns))
	for _, fn := range fns {
		fn := fn // capture loop variable
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("panic: %v", r))
					mu.Unlock()
					globalLogger.Error("parallel goroutine panic recovered",
						"panic", fmt.Sprintf("%v", r),
						"stack", string(debug.Stack()),
					)
				}
			}()
			if err := fn(); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return errors
}

// ParallelWithContext runs multiple functions concurrently with context support
// Cancels remaining goroutines on first error if cancelOnError is true
func ParallelWithContext(ctx context.Context, cancelOnError bool, fns ...func(ctx context.Context) error) error {
	if len(fns) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		firstErr error
	)

	wg.Add(len(fns))
	for _, fn := range fns {
		fn := fn
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("panic: %v", r)
						if cancelOnError {
							cancel()
						}
					})
					globalLogger.Error("parallel context goroutine panic recovered",
						"panic", fmt.Sprintf("%v", r),
						"stack", string(debug.Stack()),
					)
				}
			}()

			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := fn(ctx); err != nil {
				errOnce.Do(func() {
					firstErr = err
					if cancelOnError {
						cancel()
					}
				})
			}
		}()
	}

	wg.Wait()
	return firstErr
}

// Result holds the result of a parallel operation
type Result[T any] struct {
	Value T
	Err   error
}

// ParallelResults runs multiple functions and collects their results
func ParallelResults[T any](fns ...func() (T, error)) []Result[T] {
	results := make([]Result[T], len(fns))
	var wg sync.WaitGroup

	wg.Add(len(fns))
	for i, fn := range fns {
		i, fn := i, fn
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results[i] = Result[T]{Err: fmt.Errorf("panic: %v", r)}
					globalLogger.Error("parallel results goroutine panic recovered",
						"panic", fmt.Sprintf("%v", r),
						"stack", string(debug.Stack()),
					)
				}
			}()
			value, err := fn()
			results[i] = Result[T]{Value: value, Err: err}
		}()
	}

	wg.Wait()
	return results
}
