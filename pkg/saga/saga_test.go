package saga

import (
	"context"
	"errors"
	"testing"
)

func TestSagaSuccess(t *testing.T) {
	ctx := context.Background()
	executionOrder := []string{}

	step1 := Step{
		Name: "ReserveStock",
		Execute: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "ReserveStock")
			return nil
		},
		Compensate: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "CompensateStock")
			return nil
		},
	}

	step2 := Step{
		Name: "CapturePayment",
		Execute: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "CapturePayment")
			return nil
		},
		Compensate: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "RefundPayment")
			return nil
		},
	}

	s := New("CheckoutSaga", step1, step2)
	err := s.Run(ctx)

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if s.State != StateCompleted {
		t.Fatalf("expected state COMPLETED, got: %s", s.State)
	}
	if len(executionOrder) != 2 || executionOrder[0] != "ReserveStock" || executionOrder[1] != "CapturePayment" {
		t.Fatalf("unexpected execution order: %v", executionOrder)
	}
}

func TestSagaFailureAndCompensation(t *testing.T) {
	ctx := context.Background()
	executionOrder := []string{}

	step1 := Step{
		Name: "ReserveStock",
		Execute: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "ReserveStock")
			return nil
		},
		Compensate: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "CompensateStock")
			return nil
		},
	}

	step2 := Step{
		Name: "CapturePayment",
		Execute: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "CapturePayment")
			return errors.New("insufficient funds")
		},
		Compensate: func(ctx context.Context) error {
			executionOrder = append(executionOrder, "RefundPayment")
			return nil
		},
	}

	s := New("CheckoutSaga", step1, step2)
	err := s.Run(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s.State != StateCompensated {
		t.Fatalf("expected state COMPENSATED, got: %s", s.State)
	}
	if len(executionOrder) != 3 ||
		executionOrder[0] != "ReserveStock" ||
		executionOrder[1] != "CapturePayment" ||
		executionOrder[2] != "CompensateStock" {
		t.Fatalf("unexpected order with compensation: %v", executionOrder)
	}
}
