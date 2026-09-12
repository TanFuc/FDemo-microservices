package saga

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// State represents the lifecycle status of a Saga instance.
type State string

const (
	StatePending      State = "PENDING"
	StateExecuting    State = "EXECUTING"
	StateCompleted    State = "COMPLETED"
	StateCompensating State = "COMPENSATING"
	StateCompensated  State = "COMPENSATED"
	StateFailed       State = "FAILED"
)

// StepAction defines the execution function for a saga forward step.
type StepAction func(ctx context.Context) error

// CompensatingAction defines the rollback function when a downstream step fails.
type CompensatingAction func(ctx context.Context) error

// Step defines an individual forward and compensating step in the Saga.
type Step struct {
	Name        string
	Execute     StepAction
	Compensate  CompensatingAction
}

// Instance represents a running or finished Saga.
type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     State     `json:"state"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	Error     string    `json:"error,omitempty"`
	steps     []Step
}

// New constructs a new Saga instance with a name and defined steps.
func New(name string, steps ...Step) *Instance {
	return &Instance{
		ID:        uuid.NewString(),
		Name:      name,
		State:     StatePending,
		StartedAt: time.Now().UTC(),
		steps:     steps,
	}
}

// Run executes the saga steps sequentially. If any step fails, compensation actions
// are executed in reverse order for all completed steps.
func (s *Instance) Run(ctx context.Context) error {
	s.State = StateExecuting
	executedSteps := make([]Step, 0, len(s.steps))

	for _, step := range s.steps {
		if err := step.Execute(ctx); err != nil {
			s.State = StateCompensating
			s.Error = fmt.Sprintf("step '%s' failed: %v", step.Name, err)

			// Rollback executed steps in reverse order
			compErr := s.compensate(ctx, executedSteps)
			s.EndedAt = time.Now().UTC()

			if compErr != nil {
				s.State = StateFailed
				return fmt.Errorf("%s (compensation also failed: %v)", s.Error, compErr)
			}

			s.State = StateCompensated
			return fmt.Errorf("saga failed and compensated: %s", s.Error)
		}
		executedSteps = append(executedSteps, step)
	}

	s.State = StateCompleted
	s.EndedAt = time.Now().UTC()
	return nil
}

func (s *Instance) compensate(ctx context.Context, steps []Step) error {
	var lastErr error
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Compensate != nil {
			if err := step.Compensate(ctx); err != nil {
				lastErr = fmt.Errorf("compensate '%s' failed: %w", step.Name, err)
			}
		}
	}
	return lastErr
}
