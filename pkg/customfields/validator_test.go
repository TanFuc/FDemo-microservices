package customfields

import (
	"context"
	"errors"
	"testing"
)

func TestValidator_ValidateField(t *testing.T) {
	v := NewValidator()
	ctx := context.Background()

	// 1. Required text field test
	textField := &FieldDefinition{
		ID:         "f1",
		Name:       "username",
		FieldType:  FieldTypeText,
		IsRequired: true,
	}

	// Missing value should fail
	if err := v.ValidateField(ctx, textField, nil); err == nil {
		t.Error("expected error for missing required field, got nil")
	}
	if err := v.ValidateField(ctx, textField, ""); err == nil {
		t.Error("expected error for empty string on required field, got nil")
	}
	// Valid string should pass
	if err := v.ValidateField(ctx, textField, "valid_user"); err != nil {
		t.Errorf("unexpected error for valid string: %v", err)
	}

	// 2. Email field test
	emailField := &FieldDefinition{
		ID:        "f2",
		Name:      "email",
		FieldType: FieldTypeEmail,
	}
	if err := v.ValidateField(ctx, emailField, "invalid-email"); err == nil {
		t.Error("expected error for invalid email, got nil")
	}
	if err := v.ValidateField(ctx, emailField, "user@example.com"); err != nil {
		t.Errorf("unexpected error for valid email: %v", err)
	}

	// 3. Number field test
	numberField := &FieldDefinition{
		ID:        "f3",
		Name:      "age",
		FieldType: FieldTypeNumber,
	}
	if err := v.ValidateField(ctx, numberField, 25); err != nil {
		t.Errorf("unexpected error for valid integer: %v", err)
	}
	if err := v.ValidateField(ctx, numberField, "not-a-number"); err == nil {
		t.Error("expected error for non-number, got nil")
	}

	// 4. Boolean field test
	boolField := &FieldDefinition{
		ID:        "f4",
		Name:      "is_active",
		FieldType: FieldTypeBoolean,
	}
	if err := v.ValidateField(ctx, boolField, true); err != nil {
		t.Errorf("unexpected error for boolean true: %v", err)
	}

	// 5. Custom validator test
	errNotEven := errors.New("number must be even")
	v.RegisterCustomValidator("even_number", func(ctx context.Context, val interface{}, params map[string]interface{}) error {
		num, ok := val.(int)
		if !ok || num%2 != 0 {
			return errNotEven
		}
		return nil
	})

	fn, ok := v.GetCustomValidator("even_number")
	if !ok || fn == nil {
		t.Fatal("expected custom validator to be registered")
	}

	if err := fn(ctx, 4, nil); err != nil {
		t.Errorf("expected 4 to pass even_number validator: %v", err)
	}
	if err := fn(ctx, 5, nil); err == nil {
		t.Error("expected 5 to fail even_number validator")
	}
}
