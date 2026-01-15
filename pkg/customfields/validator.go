package customfields

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"sync"
	"time"
)

// CustomValidatorFunc is a custom validation function
type CustomValidatorFunc func(ctx context.Context, value interface{}, params map[string]interface{}) error

// Validator validates custom field values
type Validator struct {
	customValidators map[string]CustomValidatorFunc
	mu               sync.RWMutex
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	v := &Validator{
		customValidators: make(map[string]CustomValidatorFunc),
	}
	v.registerBuiltinValidators()
	return v
}

// RegisterCustomValidator registers a custom validation function
func (v *Validator) RegisterCustomValidator(name string, fn CustomValidatorFunc) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.customValidators[name] = fn
}

// GetCustomValidator returns a custom validator by name
func (v *Validator) GetCustomValidator(name string) (CustomValidatorFunc, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	fn, ok := v.customValidators[name]
	return fn, ok
}

// ValidateField validates a single field value against its definition
func (v *Validator) ValidateField(ctx context.Context, def *FieldDefinition, value interface{}) error {
	// Check required
	if def.IsRequired && (value == nil || value == "") {
		return NewFieldError(def.ID, def.Name, "field is required", ErrRequiredFieldMissing)
	}

	// If nil and not required, skip further validation
	if value == nil {
		return nil
	}

	// Type validation
	if err := v.validateType(def, value); err != nil {
		return err
	}

	// Format validation
	if err := v.validateFormat(def, value); err != nil {
		return err
	}

	// Validation rules
	if def.Validation != nil {
		if err := v.validateRules(ctx, def, value); err != nil {
			return err
		}
	}

	// Options validation (for select/multi-select)
	if err := v.validateOptions(def, value); err != nil {
		return err
	}

	return nil
}

// ValidateEntity validates all field values for an entity
func (v *Validator) ValidateEntity(ctx context.Context, definitions []*FieldDefinition, values map[string]interface{}) error {
	var errors []*FieldError

	// Create a map of definitions by ID for quick lookup
	defMap := make(map[string]*FieldDefinition)
	for _, def := range definitions {
		defMap[def.ID] = def
	}

	// Check each definition
	for _, def := range definitions {
		value := values[def.ID]
		if err := v.ValidateField(ctx, def, value); err != nil {
			if fe, ok := err.(*FieldError); ok {
				errors = append(errors, fe)
			} else {
				errors = append(errors, NewFieldError(def.ID, def.Name, err.Error(), err))
			}
		}
	}

	// Validate dependencies
	for _, def := range definitions {
		if def.DependsOn != nil {
			if err := v.validateDependency(def, defMap, values); err != nil {
				if fe, ok := err.(*FieldError); ok {
					errors = append(errors, fe)
				} else {
					errors = append(errors, NewFieldError(def.ID, def.Name, err.Error(), err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return &ValidationError{Errors: errors}
	}
	return nil
}

// validateType validates the type of the value
func (v *Validator) validateType(def *FieldDefinition, value interface{}) error {
	switch def.FieldType {
	case FieldTypeText, FieldTypeTextarea, FieldTypeEmail, FieldTypeURL, FieldTypePhone, FieldTypeRichText, FieldTypeColor:
		if _, ok := value.(string); !ok {
			return NewFieldError(def.ID, def.Name, "expected string value", ErrInvalidFieldValue)
		}
	case FieldTypeNumber:
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid
		default:
			return NewFieldError(def.ID, def.Name, "expected numeric value", ErrInvalidFieldValue)
		}
	case FieldTypeDecimal, FieldTypeCurrency, FieldTypePercentage:
		switch value.(type) {
		case float32, float64:
			// Valid
		case int, int32, int64:
			// Also valid, will be converted
		default:
			return NewFieldError(def.ID, def.Name, "expected decimal value", ErrInvalidFieldValue)
		}
	case FieldTypeBoolean, FieldTypeCheckbox:
		if _, ok := value.(bool); !ok {
			return NewFieldError(def.ID, def.Name, "expected boolean value", ErrInvalidFieldValue)
		}
	case FieldTypeDate, FieldTypeDateTime, FieldTypeTime:
		switch value.(type) {
		case time.Time, string:
			// Valid
		default:
			return NewFieldError(def.ID, def.Name, "expected date/time value", ErrInvalidFieldValue)
		}
	case FieldTypeSelect, FieldTypeRadio:
		if _, ok := value.(string); !ok {
			return NewFieldError(def.ID, def.Name, "expected string value for select", ErrInvalidFieldValue)
		}
	case FieldTypeMultiSelect:
		switch value.(type) {
		case []string, []interface{}:
			// Valid
		default:
			return NewFieldError(def.ID, def.Name, "expected array value for multi-select", ErrInvalidFieldValue)
		}
	case FieldTypeJSON, FieldTypeObject:
		switch value.(type) {
		case map[string]interface{}, JSONB:
			// Valid
		default:
			return NewFieldError(def.ID, def.Name, "expected object value", ErrInvalidFieldValue)
		}
	case FieldTypeArray:
		switch value.(type) {
		case []interface{}, []string:
			// Valid
		default:
			return NewFieldError(def.ID, def.Name, "expected array value", ErrInvalidFieldValue)
		}
	case FieldTypeRating:
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid
		default:
			return NewFieldError(def.ID, def.Name, "expected numeric rating value", ErrInvalidFieldValue)
		}
	case FieldTypeLocation:
		switch v := value.(type) {
		case map[string]interface{}:
			if _, hasLat := v["lat"]; !hasLat {
				return NewFieldError(def.ID, def.Name, "location missing lat field", ErrInvalidFieldValue)
			}
			if _, hasLng := v["lng"]; !hasLng {
				return NewFieldError(def.ID, def.Name, "location missing lng field", ErrInvalidFieldValue)
			}
		default:
			return NewFieldError(def.ID, def.Name, "expected location object with lat/lng", ErrInvalidFieldValue)
		}
	}

	return nil
}

// validateFormat validates the format of the value
func (v *Validator) validateFormat(def *FieldDefinition, value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return nil // Only validate format for strings
	}

	switch def.FieldType {
	case FieldTypeEmail:
		if _, err := mail.ParseAddress(strValue); err != nil {
			return NewFieldError(def.ID, def.Name, "invalid email format", ErrInvalidFieldValue)
		}
	case FieldTypeURL:
		if _, err := url.ParseRequestURI(strValue); err != nil {
			return NewFieldError(def.ID, def.Name, "invalid URL format", ErrInvalidFieldValue)
		}
	case FieldTypePhone:
		// Basic phone validation - allows digits, spaces, +, -, (, )
		phoneRegex := regexp.MustCompile(`^[\d\s\+\-\(\)]+$`)
		if !phoneRegex.MatchString(strValue) {
			return NewFieldError(def.ID, def.Name, "invalid phone format", ErrInvalidFieldValue)
		}
	case FieldTypeColor:
		// Validate hex color
		colorRegex := regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
		if !colorRegex.MatchString(strValue) {
			return NewFieldError(def.ID, def.Name, "invalid color format (expected #RRGGBB or #RGB)", ErrInvalidFieldValue)
		}
	case FieldTypeDate:
		if _, err := time.Parse("2006-01-02", strValue); err != nil {
			return NewFieldError(def.ID, def.Name, "invalid date format (expected YYYY-MM-DD)", ErrInvalidFieldValue)
		}
	case FieldTypeDateTime:
		if _, err := time.Parse(time.RFC3339, strValue); err != nil {
			if _, err := time.Parse("2006-01-02 15:04:05", strValue); err != nil {
				return NewFieldError(def.ID, def.Name, "invalid datetime format", ErrInvalidFieldValue)
			}
		}
	case FieldTypeTime:
		if _, err := time.Parse("15:04:05", strValue); err != nil {
			if _, err := time.Parse("15:04", strValue); err != nil {
				return NewFieldError(def.ID, def.Name, "invalid time format (expected HH:MM or HH:MM:SS)", ErrInvalidFieldValue)
			}
		}
	}

	// Check custom format if specified
	if def.Validation != nil && def.Validation.Format != nil {
		if err := v.validateCustomFormat(*def.Validation.Format, strValue); err != nil {
			return NewFieldError(def.ID, def.Name, err.Error(), ErrInvalidFieldValue)
		}
	}

	return nil
}

// validateCustomFormat validates a custom format
func (v *Validator) validateCustomFormat(format, value string) error {
	switch format {
	case "uuid":
		uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		if !uuidRegex.MatchString(value) {
			return fmt.Errorf("invalid UUID format")
		}
	case "ipv4":
		ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
		if !ipv4Regex.MatchString(value) {
			return fmt.Errorf("invalid IPv4 format")
		}
	case "ipv6":
		ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
		if !ipv6Regex.MatchString(value) {
			return fmt.Errorf("invalid IPv6 format")
		}
	case "credit_card":
		// Basic Luhn check would go here
		ccRegex := regexp.MustCompile(`^\d{13,19}$`)
		if !ccRegex.MatchString(value) {
			return fmt.Errorf("invalid credit card format")
		}
	}
	return nil
}

// validateRules validates the value against validation rules
func (v *Validator) validateRules(ctx context.Context, def *FieldDefinition, value interface{}) error {
	rules := def.Validation

	// String length validation
	if strValue, ok := value.(string); ok {
		if rules.MinLength != nil && len(strValue) < *rules.MinLength {
			return NewFieldError(def.ID, def.Name,
				fmt.Sprintf("value is too short (minimum %d characters)", *rules.MinLength),
				ErrValidationFailed)
		}
		if rules.MaxLength != nil && len(strValue) > *rules.MaxLength {
			return NewFieldError(def.ID, def.Name,
				fmt.Sprintf("value is too long (maximum %d characters)", *rules.MaxLength),
				ErrValidationFailed)
		}
	}

	// Numeric range validation
	if numValue, ok := toFloat64(value); ok {
		if rules.Min != nil && numValue < *rules.Min {
			return NewFieldError(def.ID, def.Name,
				fmt.Sprintf("value is too small (minimum %.2f)", *rules.Min),
				ErrValidationFailed)
		}
		if rules.Max != nil && numValue > *rules.Max {
			return NewFieldError(def.ID, def.Name,
				fmt.Sprintf("value is too large (maximum %.2f)", *rules.Max),
				ErrValidationFailed)
		}
	}

	// Pattern validation
	if rules.Pattern != nil {
		if strValue, ok := value.(string); ok {
			re, err := regexp.Compile(*rules.Pattern)
			if err != nil {
				return NewFieldError(def.ID, def.Name, "invalid validation pattern", ErrValidationFailed)
			}
			if !re.MatchString(strValue) {
				return NewFieldError(def.ID, def.Name, "value does not match required pattern", ErrValidationFailed)
			}
		}
	}

	// Enum validation
	if len(rules.Enum) > 0 {
		found := false
		for _, enumVal := range rules.Enum {
			if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", enumVal) {
				found = true
				break
			}
		}
		if !found {
			return NewFieldError(def.ID, def.Name, "value is not in allowed list", ErrValidationFailed)
		}
	}

	// Custom validation
	if rules.Custom != nil {
		fn, ok := v.GetCustomValidator(rules.Custom.FunctionName)
		if ok {
			if err := fn(ctx, value, rules.Custom.Parameters); err != nil {
				msg := rules.Custom.ErrorMessage
				if msg == "" {
					msg = err.Error()
				}
				return NewFieldError(def.ID, def.Name, msg, ErrValidationFailed)
			}
		}
	}

	return nil
}

// validateOptions validates the value against field options (for select/multi-select)
func (v *Validator) validateOptions(def *FieldDefinition, value interface{}) error {
	if len(def.Options) == 0 {
		return nil
	}

	// Get allowed values
	allowedValues := make(map[string]bool)
	for _, opt := range def.Options {
		allowedValues[opt.Value] = true
	}

	switch def.FieldType {
	case FieldTypeSelect, FieldTypeRadio:
		strValue, ok := value.(string)
		if !ok {
			return nil
		}
		if !allowedValues[strValue] {
			return NewFieldError(def.ID, def.Name, "value is not in allowed options", ErrValidationFailed)
		}
	case FieldTypeMultiSelect:
		var values []string
		switch v := value.(type) {
		case []string:
			values = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					values = append(values, s)
				}
			}
		default:
			return nil
		}
		for _, val := range values {
			if !allowedValues[val] {
				return NewFieldError(def.ID, def.Name,
					fmt.Sprintf("value '%s' is not in allowed options", val),
					ErrValidationFailed)
			}
		}
	}

	return nil
}

// validateDependency validates field dependencies
func (v *Validator) validateDependency(def *FieldDefinition, defMap map[string]*FieldDefinition, values map[string]interface{}) error {
	dep := def.DependsOn
	if dep == nil {
		return nil
	}

	dependentValue := values[dep.FieldID]
	conditionMet := v.evaluateDependencyCondition(dep.Condition, dependentValue, dep.Value)

	// If ShowIf is true and condition is met, the field should be shown/required
	// If ShowIf is false and condition is met, the field should be hidden
	if dep.ShowIf {
		// Field is shown when condition is met
		// If field is required but dependency not met, skip required validation
		if !conditionMet && def.IsRequired {
			// The field is hidden, so it's not required
			return nil
		}
	} else {
		// Field is hidden when condition is met
		if conditionMet && def.IsRequired {
			// The field is hidden, so it's not required
			return nil
		}
	}

	return nil
}

// evaluateDependencyCondition evaluates a dependency condition
func (v *Validator) evaluateDependencyCondition(condition DependencyCondition, value, expected interface{}) bool {
	switch condition {
	case ConditionEquals:
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case ConditionNotEquals:
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case ConditionContains:
		if strValue, ok := value.(string); ok {
			if strExpected, ok := expected.(string); ok {
				return contains(strValue, strExpected)
			}
		}
		return false
	case ConditionGreaterThan:
		valFloat, valOk := toFloat64(value)
		expFloat, expOk := toFloat64(expected)
		return valOk && expOk && valFloat > expFloat
	case ConditionLessThan:
		valFloat, valOk := toFloat64(value)
		expFloat, expOk := toFloat64(expected)
		return valOk && expOk && valFloat < expFloat
	case ConditionIn:
		if arr, ok := expected.([]interface{}); ok {
			for _, item := range arr {
				if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", item) {
					return true
				}
			}
		}
		return false
	case ConditionNotIn:
		if arr, ok := expected.([]interface{}); ok {
			for _, item := range arr {
				if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", item) {
					return false
				}
			}
		}
		return true
	case ConditionIsEmpty:
		return value == nil || value == ""
	case ConditionIsNotEmpty:
		return value != nil && value != ""
	}
	return false
}

// registerBuiltinValidators registers built-in custom validators
func (v *Validator) registerBuiltinValidators() {
	// Future date validator
	v.RegisterCustomValidator("future_date", func(ctx context.Context, value interface{}, params map[string]interface{}) error {
		var t time.Time
		switch val := value.(type) {
		case time.Time:
			t = val
		case string:
			var err error
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				t, err = time.Parse("2006-01-02", val)
				if err != nil {
					return fmt.Errorf("invalid date format")
				}
			}
		default:
			return fmt.Errorf("expected date value")
		}
		if !t.After(time.Now()) {
			return fmt.Errorf("date must be in the future")
		}
		return nil
	})

	// Past date validator
	v.RegisterCustomValidator("past_date", func(ctx context.Context, value interface{}, params map[string]interface{}) error {
		var t time.Time
		switch val := value.(type) {
		case time.Time:
			t = val
		case string:
			var err error
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				t, err = time.Parse("2006-01-02", val)
				if err != nil {
					return fmt.Errorf("invalid date format")
				}
			}
		default:
			return fmt.Errorf("expected date value")
		}
		if !t.Before(time.Now()) {
			return fmt.Errorf("date must be in the past")
		}
		return nil
	})

	// Positive number validator
	v.RegisterCustomValidator("positive", func(ctx context.Context, value interface{}, params map[string]interface{}) error {
		num, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("expected numeric value")
		}
		if num <= 0 {
			return fmt.Errorf("value must be positive")
		}
		return nil
	})

	// Non-negative number validator
	v.RegisterCustomValidator("non_negative", func(ctx context.Context, value interface{}, params map[string]interface{}) error {
		num, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("expected numeric value")
		}
		if num < 0 {
			return fmt.Errorf("value must be non-negative")
		}
		return nil
	})
}

// Helper functions

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
