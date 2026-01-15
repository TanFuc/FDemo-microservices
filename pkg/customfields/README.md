# Custom Fields Package

A flexible custom fields management system for Go microservices with permission control and validation.

## Features

- **20+ Field Types**: Text, number, date, select, multi-select, JSON, and more
- **Validation Engine**: Required, patterns, ranges, custom validators
- **EAV Storage**: Efficient Entity-Attribute-Value pattern with typed columns
- **Caching**: Redis-based field definition and value caching
- **Permissions**: Field-level access control integration
- **Query Builder**: Search entities by custom field values
- **Bulk Operations**: Efficient batch get/set operations
- **Import/Export**: JSON-based definition import/export
- **JSON Schema**: Automatic JSON Schema generation

## Installation

```bash
go get microservices/pkg/customfields
```

## Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "microservices/pkg/customfields"
    _ "microservices/pkg/customfields/storage" // Register storage implementation
)

func main() {
    // Create configuration
    cfg := &customfields.Config{
        Database: &customfields.DatabaseConfig{
            Host:     "localhost",
            Port:     5432,
            User:     "postgres",
            Password: "password",
            Database: "customfields",
        },
        Cache: &customfields.CacheConfig{
            Enabled: true,
            Type:    "redis",
            Redis: &customfields.RedisConfig{
                Addr: "localhost:6379",
            },
        },
    }

    // Create field manager
    fm, err := customfields.New(cfg)
    if err != nil {
        panic(err)
    }
    defer fm.Close()

    ctx := context.Background()

    // Create a custom field definition
    field := &customfields.FieldDefinition{
        EntityType:  "order",
        Name:        "priority",
        Label:       "Priority Level",
        FieldType:   customfields.FieldTypeSelect,
        DataType:    customfields.DataTypeString,
        IsRequired:  true,
        IsSearchable: true,
        Options: []*customfields.FieldOption{
            {Value: "low", Label: "Low"},
            {Value: "medium", Label: "Medium", IsDefault: true},
            {Value: "high", Label: "High"},
        },
    }
    fm.CreateFieldDefinition(ctx, field)

    // Set field value for an entity
    fm.SetFieldValue(ctx, "order", "order-123", map[string]interface{}{
        field.ID: "high",
    })

    // Get field values
    values, _ := fm.GetAllFieldValues(ctx, "order", "order-123")
    fmt.Printf("Priority: %v\n", values[field.ID])
}
```

### Field Types

```go
// Text fields
FieldTypeText        // Single line text
FieldTypeTextarea    // Multi-line text
FieldTypeRichText    // Rich text / HTML

// Numeric fields
FieldTypeNumber      // Integer
FieldTypeDecimal     // Decimal/float
FieldTypeCurrency    // Money
FieldTypePercentage  // Percentage
FieldTypeRating      // Star rating (0-5)

// Date/Time fields
FieldTypeDate        // Date only
FieldTypeDateTime    // Date and time
FieldTypeTime        // Time only

// Selection fields
FieldTypeSelect      // Single select dropdown
FieldTypeMultiSelect // Multiple selection
FieldTypeRadio       // Radio buttons
FieldTypeCheckbox    // Checkboxes

// Special fields
FieldTypeEmail       // Email address
FieldTypeURL         // URL
FieldTypePhone       // Phone number
FieldTypeColor       // Color picker (#RRGGBB)
FieldTypeLocation    // Lat/Lng coordinates
FieldTypeJSON        // JSON object
FieldTypeArray       // Array
FieldTypeReference   // Foreign key reference
FieldTypeFile        // File upload
FieldTypeImage       // Image upload
FieldTypeFormula     // Computed field
```

### Validation Rules

```go
field := &customfields.FieldDefinition{
    EntityType: "product",
    Name:       "sku",
    FieldType:  customfields.FieldTypeText,
    DataType:   customfields.DataTypeString,
    IsRequired: true,
    IsUnique:   true,
    Validation: &customfields.ValidationRules{
        MinLength: intPtr(5),
        MaxLength: intPtr(20),
        Pattern:   strPtr(`^[A-Z0-9-]+$`),
    },
}

// Numeric validation
priceField := &customfields.FieldDefinition{
    EntityType: "product",
    Name:       "price",
    FieldType:  customfields.FieldTypeDecimal,
    DataType:   customfields.DataTypeFloat,
    Validation: &customfields.ValidationRules{
        Min: floatPtr(0),
        Max: floatPtr(999999.99),
    },
}

// Custom validation
field.Validation.Custom = &customfields.CustomValidation{
    FunctionName: "future_date",
    ErrorMessage: "Date must be in the future",
}
```

### Field Dependencies

```go
// Show field only when another field has specific value
shippingAddressField := &customfields.FieldDefinition{
    EntityType: "order",
    Name:       "shipping_address",
    FieldType:  customfields.FieldTypeTextarea,
    DependsOn: &customfields.FieldDependency{
        FieldID:   deliveryMethodField.ID,
        Condition: customfields.ConditionEquals,
        Value:     "delivery",
        ShowIf:    true,
    },
}
```

### Field Permissions

```go
// Restrict field access by roles
field := &customfields.FieldDefinition{
    EntityType: "order",
    Name:       "internal_notes",
    FieldType:  customfields.FieldTypeTextarea,
    Permissions: &customfields.FieldPermissions{
        ReadRoles:  []string{"admin", "manager"},
        WriteRoles: []string{"admin"},
    },
}
```

### Search by Field Values

```go
// Find orders with high priority
results, _ := fm.SearchByFieldValue(ctx, &customfields.FieldQuery{
    EntityType: "order",
    Conditions: []*customfields.QueryCondition{
        {
            FieldID:  priorityField.ID,
            Operator: customfields.ConditionEquals,
            Value:    "high",
        },
    },
    Limit:  100,
    Offset: 0,
})

for _, result := range results {
    fmt.Printf("Order: %s\n", result.EntityID)
}
```

### Bulk Operations

```go
// Set values for multiple entities
operations := []*customfields.BulkFieldOperation{
    {
        EntityType: "order",
        EntityID:   "order-1",
        Values:     map[string]interface{}{"status": "shipped"},
    },
    {
        EntityType: "order",
        EntityID:   "order-2",
        Values:     map[string]interface{}{"status": "processing"},
    },
}
fm.BulkSetFieldValues(ctx, operations)

// Get values for multiple entities
valuesMap, _ := fm.BulkGetFieldValues(ctx, "order", []string{"order-1", "order-2", "order-3"})
```

### JSON Schema Generation

```go
// Generate JSON Schema for entity type
schema, _ := fm.GenerateJSONSchema(ctx, "order")
// Returns standard JSON Schema draft-07 format
```

### Import/Export

```go
// Export field definitions
data, _ := fm.ExportFieldDefinitions(ctx, "order")
// Save to file...

// Import field definitions
fm.ImportFieldDefinitions(ctx, data)
```

## Configuration

```go
type Config struct {
    // Database configuration
    Database *DatabaseConfig

    // Redis cache configuration
    Cache *CacheConfig

    // Multi-tenancy default
    DefaultTenantID string

    // Performance settings
    BatchSize    int
    QueryTimeout time.Duration

    // Features
    EnableAuditLog   bool
    EnableEncryption bool
}
```

## Database Schema

The package uses an EAV (Entity-Attribute-Value) pattern with typed columns for efficient storage and querying:

```sql
-- Field definitions
CREATE TABLE custom_field_definitions (
    id VARCHAR(100) PRIMARY KEY,
    entity_type VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    field_type VARCHAR(50) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    validation_rules JSONB,
    options JSONB,
    permissions JSONB,
    ...
);

-- Field values with typed columns
CREATE TABLE custom_field_values (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    field_id VARCHAR(100) NOT NULL,
    value_string TEXT,
    value_int BIGINT,
    value_float DOUBLE PRECISION,
    value_bool BOOLEAN,
    value_date TIMESTAMP,
    value_json JSONB,
    ...
);
```

## Performance

- **Typed columns**: Efficient querying by data type
- **Caching**: Field definitions and values cached in Redis
- **Bulk operations**: Batch get/set for multiple entities
- **Indexes**: Searchable fields are automatically indexed

## Testing

```bash
go test ./pkg/customfields/...
```

## License

MIT
