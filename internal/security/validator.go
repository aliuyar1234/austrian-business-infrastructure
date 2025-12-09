package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

var (
	// ErrInvalidJSONStructure indicates the AI output is not valid JSON
	ErrInvalidJSONStructure = errors.New("invalid JSON structure in AI output")
	// ErrUnexpectedFields indicates unexpected fields in AI output
	ErrUnexpectedFields = errors.New("unexpected fields in AI output")
	// ErrMissingRequiredFields indicates required fields are missing
	ErrMissingRequiredFields = errors.New("required fields missing in AI output")
	// ErrFieldTypeMismatch indicates a field has wrong type
	ErrFieldTypeMismatch = errors.New("field type mismatch in AI output")
)

// OutputValidator validates AI output against expected schemas
type OutputValidator struct {
	allowExtraFields bool
}

// NewOutputValidator creates a new output validator
func NewOutputValidator() *OutputValidator {
	return &OutputValidator{
		allowExtraFields: false, // Strict by default
	}
}

// WithAllowExtraFields sets whether to allow extra fields in output
func (v *OutputValidator) WithAllowExtraFields(allow bool) *OutputValidator {
	v.allowExtraFields = allow
	return v
}

// ValidationResult contains the result of output validation
type ValidationResult struct {
	Valid            bool
	Errors           []string
	StrippedFields   []string
	SanitizedOutput  interface{}
}

// Schema defines expected fields and their types for validation
type Schema struct {
	Fields   map[string]FieldDef
	Required []string
}

// FieldDef defines a field's expected type and constraints
type FieldDef struct {
	Type     string // "string", "number", "boolean", "array", "object"
	Required bool
	MaxLen   int    // For strings
	MinLen   int    // For strings
	Nested   *Schema // For objects
	Items    *FieldDef // For arrays
}

// ValidateJSON validates JSON output against a schema
func (v *OutputValidator) ValidateJSON(output []byte, schema *Schema) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Parse JSON
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON: %v", err))
		return result
	}

	// Validate against schema
	v.validateObject(data, schema, "", result)

	// Strip unexpected fields if not allowed
	if !v.allowExtraFields {
		v.stripUnexpectedFields(data, schema, result)
	}

	result.SanitizedOutput = data
	return result
}

// validateObject validates a map against a schema
func (v *OutputValidator) validateObject(data map[string]interface{}, schema *Schema, path string, result *ValidationResult) {
	if schema == nil {
		return
	}

	// Check required fields
	for _, reqField := range schema.Required {
		if _, exists := data[reqField]; !exists {
			result.Valid = false
			fieldPath := joinPath(path, reqField)
			result.Errors = append(result.Errors, fmt.Sprintf("missing required field: %s", fieldPath))
		}
	}

	// Validate each field
	for fieldName, value := range data {
		fieldPath := joinPath(path, fieldName)
		fieldDef, defined := schema.Fields[fieldName]

		if !defined {
			if !v.allowExtraFields {
				result.StrippedFields = append(result.StrippedFields, fieldPath)
			}
			continue
		}

		v.validateField(value, &fieldDef, fieldPath, result)
	}
}

// validateField validates a single field value
func (v *OutputValidator) validateField(value interface{}, def *FieldDef, path string, result *ValidationResult) {
	if value == nil {
		if def.Required {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("required field is null: %s", path))
		}
		return
	}

	// Check type
	actualType := getJSONType(value)
	if def.Type != "" && actualType != def.Type {
		// Allow number to be int or float
		if !(def.Type == "number" && (actualType == "int" || actualType == "float")) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("type mismatch at %s: expected %s, got %s", path, def.Type, actualType))
			return
		}
	}

	// Type-specific validation
	switch def.Type {
	case "string":
		str, ok := value.(string)
		if ok {
			if def.MaxLen > 0 && len(str) > def.MaxLen {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("string too long at %s: max %d, got %d", path, def.MaxLen, len(str)))
			}
			if def.MinLen > 0 && len(str) < def.MinLen {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("string too short at %s: min %d, got %d", path, def.MinLen, len(str)))
			}
		}

	case "object":
		if obj, ok := value.(map[string]interface{}); ok && def.Nested != nil {
			v.validateObject(obj, def.Nested, path, result)
		}

	case "array":
		if arr, ok := value.([]interface{}); ok && def.Items != nil {
			for i, item := range arr {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				v.validateField(item, def.Items, itemPath, result)
			}
		}
	}
}

// stripUnexpectedFields removes fields not in schema
func (v *OutputValidator) stripUnexpectedFields(data map[string]interface{}, schema *Schema, result *ValidationResult) {
	if schema == nil {
		return
	}

	toDelete := make([]string, 0)
	for fieldName, value := range data {
		fieldDef, defined := schema.Fields[fieldName]
		if !defined {
			toDelete = append(toDelete, fieldName)
			continue
		}

		// Recursively strip nested objects
		if fieldDef.Type == "object" && fieldDef.Nested != nil {
			if obj, ok := value.(map[string]interface{}); ok {
				v.stripUnexpectedFields(obj, fieldDef.Nested, result)
			}
		}

		// Strip from arrays
		if fieldDef.Type == "array" && fieldDef.Items != nil && fieldDef.Items.Type == "object" && fieldDef.Items.Nested != nil {
			if arr, ok := value.([]interface{}); ok {
				for _, item := range arr {
					if obj, ok := item.(map[string]interface{}); ok {
						v.stripUnexpectedFields(obj, fieldDef.Items.Nested, result)
					}
				}
			}
		}
	}

	// Delete unexpected fields
	for _, field := range toDelete {
		delete(data, field)
	}
}

// getJSONType returns the JSON type name for a value
func getJSONType(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Float64:
		return "number"
	case reflect.Int, reflect.Int64:
		return "int"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "unknown"
	}
}

// joinPath joins path segments
func joinPath(base, field string) string {
	if base == "" {
		return field
	}
	return base + "." + field
}

// Common schemas for AI responses

// DocumentAnalysisSchema is the expected schema for document analysis responses
var DocumentAnalysisSchema = &Schema{
	Required: []string{"summary", "document_type"},
	Fields: map[string]FieldDef{
		"summary": {
			Type:   "string",
			MaxLen: 5000,
		},
		"document_type": {
			Type:   "string",
			MaxLen: 100,
		},
		"deadline": {
			Type:   "string",
			MaxLen: 100,
		},
		"amount": {
			Type: "number",
		},
		"action_items": {
			Type: "array",
			Items: &FieldDef{
				Type:   "string",
				MaxLen: 500,
			},
		},
		"confidence": {
			Type: "number",
		},
	},
}

// ActionItemsSchema is the expected schema for action items extraction
var ActionItemsSchema = &Schema{
	Required: []string{"items"},
	Fields: map[string]FieldDef{
		"items": {
			Type: "array",
			Items: &FieldDef{
				Type: "object",
				Nested: &Schema{
					Required: []string{"description"},
					Fields: map[string]FieldDef{
						"description": {Type: "string", MaxLen: 1000},
						"deadline":    {Type: "string", MaxLen: 50},
						"priority":    {Type: "string", MaxLen: 20},
					},
				},
			},
		},
	},
}
