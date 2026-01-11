package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	Jitter         bool
}

// DefaultConfig returns default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// Do executes the given function with exponential backoff retry
func Do(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Execute the function
		if err := fn(); err != nil {
			lastErr = err

			// If we've exhausted retries, return the error
			if attempt >= cfg.MaxRetries {
				return lastErr
			}

			// Calculate delay with exponential backoff
			delay := cfg.InitialDelay * time.Duration(math.Pow(cfg.BackoffFactor, float64(attempt)))

			// Cap at max delay
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			// Add jitter if enabled (0-25% random variation)
			if cfg.Jitter {
				jitter := time.Duration(rand.Float64() * 0.25 * float64(delay))
				delay += jitter
			}

			// Wait or check for context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Success
			return nil
		}
	}

	return lastErr
}

// DoWithResult executes the given function with retry and returns the result
func DoWithResult[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Execute the function
		res, err := fn()
		if err != nil {
			lastErr = err

			// If we've exhausted retries, return the error
			if attempt >= cfg.MaxRetries {
				return result, lastErr
			}

			// Calculate delay with exponential backoff
			delay := cfg.InitialDelay * time.Duration(math.Pow(cfg.BackoffFactor, float64(attempt)))

			// Cap at max delay
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			// Add jitter if enabled
			if cfg.Jitter {
				jitter := time.Duration(rand.Float64() * 0.25 * float64(delay))
				delay += jitter
			}

			// Wait or check for context cancellation
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Success
			return res, nil
		}
	}

	return result, lastErr
}
