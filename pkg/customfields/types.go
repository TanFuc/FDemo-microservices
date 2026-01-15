package customfields

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// FieldType represents the type of a custom field
type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeTextarea    FieldType = "textarea"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDecimal     FieldType = "decimal"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeDate        FieldType = "date"
	FieldTypeDateTime    FieldType = "datetime"
	FieldTypeTime        FieldType = "time"
	FieldTypeEmail       FieldType = "email"
	FieldTypeURL         FieldType = "url"
	FieldTypePhone       FieldType = "phone"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiSelect FieldType = "multi_select"
	FieldTypeRadio       FieldType = "radio"
	FieldTypeCheckbox    FieldType = "checkbox"
	FieldTypeFile        FieldType = "file"
	FieldTypeImage       FieldType = "image"
	FieldTypeJSON        FieldType = "json"
	FieldTypeArray       FieldType = "array"
	FieldTypeObject      FieldType = "object"
	FieldTypeReference   FieldType = "reference"
	FieldTypeFormula     FieldType = "formula"
	FieldTypeRichText    FieldType = "richtext"
	FieldTypeColor       FieldType = "color"
	FieldTypeCurrency    FieldType = "currency"
	FieldTypePercentage  FieldType = "percentage"
	FieldTypeRating      FieldType = "rating"
	FieldTypeLocation    FieldType = "location"
)

// DataType represents the underlying data type for storage
type DataType string

const (
	DataTypeString DataType = "string"
	DataTypeInt    DataType = "int"
	DataTypeFloat  DataType = "float"
	DataTypeBool   DataType = "bool"
	DataTypeDate   DataType = "date"
	DataTypeJSON   DataType = "json"
	DataTypeBinary DataType = "binary"
)

// DependencyCondition represents the condition type for field dependencies
type DependencyCondition string

const (
	ConditionEquals      DependencyCondition = "equals"
	ConditionNotEquals   DependencyCondition = "not_equals"
	ConditionContains    DependencyCondition = "contains"
	ConditionGreaterThan DependencyCondition = "gt"
	ConditionLessThan    DependencyCondition = "lt"
	ConditionIn          DependencyCondition = "in"
	ConditionNotIn       DependencyCondition = "not_in"
	ConditionIsEmpty     DependencyCondition = "is_empty"
	ConditionIsNotEmpty  DependencyCondition = "is_not_empty"
)

// Standard errors
var (
	ErrFieldNotFound         = errors.New("customfields: field not found")
	ErrFieldAlreadyExists    = errors.New("customfields: field already exists")
	ErrInvalidFieldType      = errors.New("customfields: invalid field type")
	ErrInvalidFieldValue     = errors.New("customfields: invalid field value")
	ErrValidationFailed      = errors.New("customfields: validation failed")
	ErrRequiredFieldMissing  = errors.New("customfields: required field missing")
	ErrUniqueConstraint      = errors.New("customfields: unique constraint violated")
	ErrDependencyNotMet      = errors.New("customfields: field dependency not met")
	ErrDatabaseOperation     = errors.New("customfields: database operation failed")
	ErrCacheOperation        = errors.New("customfields: cache operation failed")
	ErrInvalidConfig         = errors.New("customfields: invalid configuration")
	ErrConnectionFailed      = errors.New("customfields: connection failed")
	ErrPermissionDenied      = errors.New("customfields: permission denied")
	ErrEntityNotFound        = errors.New("customfields: entity not found")
	ErrInvalidInput          = errors.New("customfields: invalid input")
)

// FieldError represents a detailed field error
type FieldError struct {
	FieldID string // Field ID that caused the error
	Field   string // Field name
	Message string // Error message
	Err     error  // Underlying error
}

func (e *FieldError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("customfields: field '%s': %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("customfields: field '%s': %s", e.Field, e.Message)
}

func (e *FieldError) Unwrap() error {
	return e.Err
}

// NewFieldError creates a new FieldError
func NewFieldError(fieldID, fieldName, message string, err error) *FieldError {
	return &FieldError{
		FieldID: fieldID,
		Field:   fieldName,
		Message: message,
		Err:     err,
	}
}

// ValidationError represents multiple validation errors
type ValidationError struct {
	Errors []*FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "customfields: validation failed"
	}
	return fmt.Sprintf("customfields: validation failed with %d errors: %s", len(e.Errors), e.Errors[0].Error())
}

// FieldDefinition represents a custom field definition
type FieldDefinition struct {
	ID           string    `json:"id" gorm:"primaryKey;size:100"`
	EntityType   string    `json:"entity_type" gorm:"size:100;not null;index:idx_entity_type"`
	Name         string    `json:"name" gorm:"size:100;not null"`
	Label        string    `json:"label" gorm:"size:200"`
	Description  string    `json:"description" gorm:"size:500"`
	FieldType    FieldType `json:"field_type" gorm:"size:50;not null"`
	DataType     DataType  `json:"data_type" gorm:"size:50;not null"`
	IsRequired   bool      `json:"is_required" gorm:"default:false"`
	IsUnique     bool      `json:"is_unique" gorm:"default:false"`
	IsSearchable bool      `json:"is_searchable" gorm:"default:false;index:idx_searchable"`
	IsSortable   bool      `json:"is_sortable" gorm:"default:false"`

	// Validation
	Validation     *ValidationRules `json:"validation" gorm:"-"`
	ValidationJSON string           `json:"-" gorm:"column:validation_rules;type:jsonb"`

	// Default Value
	DefaultValue     interface{} `json:"default_value" gorm:"-"`
	DefaultValueJSON string      `json:"-" gorm:"column:default_value;type:jsonb"`

	// Options for select/multi-select
	Options     []*FieldOption `json:"options" gorm:"-"`
	OptionsJSON string         `json:"-" gorm:"column:options;type:jsonb"`

	// Dependencies
	DependsOn     *FieldDependency `json:"depends_on" gorm:"-"`
	DependsOnJSON string           `json:"-" gorm:"column:depends_on;type:jsonb"`

	// Display
	DisplayOrder int    `json:"display_order" gorm:"default:0"`
	DisplayGroup string `json:"display_group" gorm:"size:100"`
	Placeholder  string `json:"placeholder" gorm:"size:200"`
	HelpText     string `json:"help_text" gorm:"size:500"`

	// Permissions
	Permissions     *FieldPermissions `json:"permissions" gorm:"-"`
	PermissionsJSON string            `json:"-" gorm:"column:permissions;type:jsonb"`

	// Storage optimization
	Indexed   bool `json:"indexed" gorm:"default:false"`
	Encrypted bool `json:"encrypted" gorm:"default:false"`

	// Metadata
	Metadata     map[string]interface{} `json:"metadata" gorm:"-"`
	MetadataJSON string                 `json:"-" gorm:"column:metadata;type:jsonb"`

	TenantID  string    `json:"tenant_id" gorm:"size:100;index:idx_tenant"`
	Version   int       `json:"version" gorm:"default:1"`
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the table name for FieldDefinition
func (FieldDefinition) TableName() string {
	return "custom_field_definitions"
}

// BeforeSave serializes JSON fields before saving
func (fd *FieldDefinition) BeforeSave() error {
	if fd.Validation != nil {
		data, err := json.Marshal(fd.Validation)
		if err != nil {
			return err
		}
		fd.ValidationJSON = string(data)
	}
	if fd.DefaultValue != nil {
		data, err := json.Marshal(fd.DefaultValue)
		if err != nil {
			return err
		}
		fd.DefaultValueJSON = string(data)
	}
	if fd.Options != nil {
		data, err := json.Marshal(fd.Options)
		if err != nil {
			return err
		}
		fd.OptionsJSON = string(data)
	}
	if fd.DependsOn != nil {
		data, err := json.Marshal(fd.DependsOn)
		if err != nil {
			return err
		}
		fd.DependsOnJSON = string(data)
	}
	if fd.Permissions != nil {
		data, err := json.Marshal(fd.Permissions)
		if err != nil {
			return err
		}
		fd.PermissionsJSON = string(data)
	}
	if fd.Metadata != nil {
		data, err := json.Marshal(fd.Metadata)
		if err != nil {
			return err
		}
		fd.MetadataJSON = string(data)
	}
	return nil
}

// AfterFind deserializes JSON fields after finding
func (fd *FieldDefinition) AfterFind() error {
	if fd.ValidationJSON != "" {
		fd.Validation = &ValidationRules{}
		if err := json.Unmarshal([]byte(fd.ValidationJSON), fd.Validation); err != nil {
			return err
		}
	}
	if fd.DefaultValueJSON != "" {
		if err := json.Unmarshal([]byte(fd.DefaultValueJSON), &fd.DefaultValue); err != nil {
			return err
		}
	}
	if fd.OptionsJSON != "" {
		if err := json.Unmarshal([]byte(fd.OptionsJSON), &fd.Options); err != nil {
			return err
		}
	}
	if fd.DependsOnJSON != "" {
		fd.DependsOn = &FieldDependency{}
		if err := json.Unmarshal([]byte(fd.DependsOnJSON), fd.DependsOn); err != nil {
			return err
		}
	}
	if fd.PermissionsJSON != "" {
		fd.Permissions = &FieldPermissions{}
		if err := json.Unmarshal([]byte(fd.PermissionsJSON), fd.Permissions); err != nil {
			return err
		}
	}
	if fd.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(fd.MetadataJSON), &fd.Metadata); err != nil {
			return err
		}
	}
	return nil
}

// ValidationRules defines validation rules for a field
type ValidationRules struct {
	MinLength *int           `json:"min_length,omitempty"`
	MaxLength *int           `json:"max_length,omitempty"`
	Min       *float64       `json:"min,omitempty"`
	Max       *float64       `json:"max,omitempty"`
	Pattern   *string        `json:"pattern,omitempty"` // Regex pattern
	Format    *string        `json:"format,omitempty"`  // email, url, etc.
	Enum      []interface{}  `json:"enum,omitempty"`
	Custom    *CustomValidation `json:"custom,omitempty"`
}

// CustomValidation represents a custom validation function
type CustomValidation struct {
	FunctionName string                 `json:"function_name"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// FieldOption represents an option for select/multi-select fields
type FieldOption struct {
	Value        string `json:"value"`
	Label        string `json:"label"`
	Color        string `json:"color,omitempty"`
	Icon         string `json:"icon,omitempty"`
	IsDefault    bool   `json:"is_default,omitempty"`
	DisplayOrder int    `json:"display_order,omitempty"`
}

// FieldDependency represents a field dependency configuration
type FieldDependency struct {
	FieldID   string              `json:"field_id"`
	Condition DependencyCondition `json:"condition"`
	Value     interface{}         `json:"value"`
	ShowIf    bool                `json:"show_if"` // true = show if condition met, false = hide
}

// FieldPermissions represents field-level permissions
type FieldPermissions struct {
	ReadRoles  []string          `json:"read_roles,omitempty"`
	WriteRoles []string          `json:"write_roles,omitempty"`
	ViewPolicy *PermissionPolicy `json:"view_policy,omitempty"`
	EditPolicy *PermissionPolicy `json:"edit_policy,omitempty"`
}

// PermissionPolicy represents a permission policy for a field
type PermissionPolicy struct {
	Type       string   `json:"type"` // "role", "attribute", "custom"
	Rules      []string `json:"rules,omitempty"`
	CustomFunc *string  `json:"custom_func,omitempty"`
}

// FieldFilter represents filters for querying field definitions
type FieldFilter struct {
	EntityType   *string    `json:"entity_type"`
	FieldType    *FieldType `json:"field_type"`
	Name         *string    `json:"name"`
	IsRequired   *bool      `json:"is_required"`
	IsSearchable *bool      `json:"is_searchable"`
	DisplayGroup *string    `json:"display_group"`
	TenantID     *string    `json:"tenant_id"`
	Limit        int        `json:"limit"`
	Offset       int        `json:"offset"`
}

// FieldValue represents a custom field value
type FieldValue struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	EntityType   string    `json:"entity_type" gorm:"size:100;not null;index:idx_entity"`
	EntityID     string    `json:"entity_id" gorm:"size:100;not null;index:idx_entity"`
	FieldID      string    `json:"field_id" gorm:"size:100;not null;index:idx_field"`

	// Multiple columns for different data types (optimization)
	ValueString *string    `json:"value_string,omitempty" gorm:"column:value_string;type:text"`
	ValueInt    *int64     `json:"value_int,omitempty" gorm:"column:value_int"`
	ValueFloat  *float64   `json:"value_float,omitempty" gorm:"column:value_float"`
	ValueBool   *bool      `json:"value_bool,omitempty" gorm:"column:value_bool"`
	ValueDate   *time.Time `json:"value_date,omitempty" gorm:"column:value_date"`
	ValueJSON   JSONB      `json:"value_json,omitempty" gorm:"column:value_json;type:jsonb"`
	ValueBinary []byte     `json:"value_binary,omitempty" gorm:"column:value_binary"`

	DisplayValue string    `json:"display_value,omitempty" gorm:"size:500"`
	TenantID     string    `json:"tenant_id" gorm:"size:100;index:idx_tenant"`
	UpdatedBy    string    `json:"updated_by" gorm:"size:100"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName returns the table name for FieldValue
func (FieldValue) TableName() string {
	return "custom_field_values"
}

// JSONB represents a JSONB type for PostgreSQL
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for JSONB: %T", value)
	}

	return json.Unmarshal(data, j)
}

// GetValue returns the actual value based on data type
func (fv *FieldValue) GetValue(dataType DataType) interface{} {
	switch dataType {
	case DataTypeString:
		if fv.ValueString != nil {
			return *fv.ValueString
		}
	case DataTypeInt:
		if fv.ValueInt != nil {
			return *fv.ValueInt
		}
	case DataTypeFloat:
		if fv.ValueFloat != nil {
			return *fv.ValueFloat
		}
	case DataTypeBool:
		if fv.ValueBool != nil {
			return *fv.ValueBool
		}
	case DataTypeDate:
		if fv.ValueDate != nil {
			return *fv.ValueDate
		}
	case DataTypeJSON:
		return fv.ValueJSON
	case DataTypeBinary:
		return fv.ValueBinary
	}
	return nil
}

// SetValue sets the value based on data type
func (fv *FieldValue) SetValue(value interface{}, dataType DataType) error {
	// Clear all values first
	fv.ValueString = nil
	fv.ValueInt = nil
	fv.ValueFloat = nil
	fv.ValueBool = nil
	fv.ValueDate = nil
	fv.ValueJSON = nil
	fv.ValueBinary = nil

	if value == nil {
		return nil
	}

	switch dataType {
	case DataTypeString:
		s := fmt.Sprintf("%v", value)
		fv.ValueString = &s
	case DataTypeInt:
		switch v := value.(type) {
		case int:
			i := int64(v)
			fv.ValueInt = &i
		case int64:
			fv.ValueInt = &v
		case float64:
			i := int64(v)
			fv.ValueInt = &i
		default:
			return fmt.Errorf("cannot convert %T to int", value)
		}
	case DataTypeFloat:
		switch v := value.(type) {
		case float64:
			fv.ValueFloat = &v
		case float32:
			f := float64(v)
			fv.ValueFloat = &f
		case int:
			f := float64(v)
			fv.ValueFloat = &f
		case int64:
			f := float64(v)
			fv.ValueFloat = &f
		default:
			return fmt.Errorf("cannot convert %T to float", value)
		}
	case DataTypeBool:
		switch v := value.(type) {
		case bool:
			fv.ValueBool = &v
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}
	case DataTypeDate:
		switch v := value.(type) {
		case time.Time:
			fv.ValueDate = &v
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				t, err = time.Parse("2006-01-02", v)
				if err != nil {
					return fmt.Errorf("cannot parse date: %v", err)
				}
			}
			fv.ValueDate = &t
		default:
			return fmt.Errorf("cannot convert %T to date", value)
		}
	case DataTypeJSON:
		switch v := value.(type) {
		case map[string]interface{}:
			fv.ValueJSON = JSONB(v)
		case JSONB:
			fv.ValueJSON = v
		default:
			// Try to convert to JSON
			data, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("cannot convert %T to JSON: %v", value, err)
			}
			var m map[string]interface{}
			if err := json.Unmarshal(data, &m); err != nil {
				return fmt.Errorf("cannot unmarshal to JSON: %v", err)
			}
			fv.ValueJSON = JSONB(m)
		}
	case DataTypeBinary:
		switch v := value.(type) {
		case []byte:
			fv.ValueBinary = v
		case string:
			fv.ValueBinary = []byte(v)
		default:
			return fmt.Errorf("cannot convert %T to binary", value)
		}
	}

	return nil
}

// EntityFieldValues represents all field values for an entity
type EntityFieldValues struct {
	EntityType string                   `json:"entity_type"`
	EntityID   string                   `json:"entity_id"`
	Fields     map[string]*FieldValue   `json:"fields"`
	TenantID   string                   `json:"tenant_id"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

// BulkFieldOperation represents a bulk field operation
type BulkFieldOperation struct {
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Values     map[string]interface{} `json:"values"`
}

// FieldQuery represents a query for searching by field values
type FieldQuery struct {
	EntityType  string         `json:"entity_type"`
	Conditions  []*QueryCondition `json:"conditions"`
	OrderBy     string         `json:"order_by,omitempty"`
	OrderDir    string         `json:"order_dir,omitempty"` // "asc" or "desc"
	Limit       int            `json:"limit,omitempty"`
	Offset      int            `json:"offset,omitempty"`
	TenantID    *string        `json:"tenant_id,omitempty"`
}

// QueryCondition represents a condition in a field query
type QueryCondition struct {
	FieldID   string              `json:"field_id"`
	Operator  DependencyCondition `json:"operator"`
	Value     interface{}         `json:"value"`
	LogicalOp string              `json:"logical_op,omitempty"` // "AND" or "OR"
}

// EntitySchema represents the schema for an entity type
type EntitySchema struct {
	EntityType string             `json:"entity_type"`
	Fields     []*FieldDefinition `json:"fields"`
	Version    int                `json:"version"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// FieldAuditLog represents an audit log entry for field changes
type FieldAuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	EntityType string    `json:"entity_type" gorm:"size:100;index:idx_entity_audit"`
	EntityID   string    `json:"entity_id" gorm:"size:100;index:idx_entity_audit"`
	FieldID    string    `json:"field_id" gorm:"size:100"`
	Action     string    `json:"action" gorm:"size:50"` // create, update, delete
	OldValue   JSONB     `json:"old_value,omitempty" gorm:"type:jsonb"`
	NewValue   JSONB     `json:"new_value,omitempty" gorm:"type:jsonb"`
	ChangedBy  string    `json:"changed_by" gorm:"size:100"`
	ChangedAt  time.Time `json:"changed_at" gorm:"index:idx_changed_at"`
	TenantID   string    `json:"tenant_id" gorm:"size:100"`
}

// TableName returns the table name for FieldAuditLog
func (FieldAuditLog) TableName() string {
	return "custom_field_audit_log"
}
