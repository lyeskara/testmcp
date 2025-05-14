package converter

import (
	"encoding/json"
	"fmt"
)

// GenerateJSONSchemaDraft7 converts a slice of Arg structs into a JSON Schema Draft 7 string.
// It creates a root object schema with properties for each argument.
func GenerateJSONSchemaDraft7(args []Arg) (string, error) {
	rootSchema := map[string]interface{}{
		"type": "object",
	}

	properties := make(map[string]interface{})
	requiredProperties := []string{}

	for _, arg := range args {
		if arg.Deprecated {
			continue
		}

		var propSchemaMap map[string]interface{}
		var err error

		if arg.Source == "body" {
			if len(arg.ContentTypes) == 0 {
				continue
			} else if len(arg.ContentTypes) == 1 {
				for _, schema := range arg.ContentTypes {
					propSchemaMap, err = schemaToDraft7Map(schema) // Assign to err
					if err != nil {
						return "", fmt.Errorf("failed to convert body schema: %w", err)
					}
				}
			} else {
				// Multiple content types for the body - create a oneOf composition
				oneOfDraft7 := make([]map[string]interface{}, 0, len(arg.ContentTypes))
				for contentType, schema := range arg.ContentTypes {
					branchSchema, err := schemaToDraft7Map(schema) // Assign to err
					if err != nil {
						return "", fmt.Errorf("failed to convert body schema branch for content type '%s': %w", contentType, err)
					}
					if branchSchema != nil {
						// Optionally add content type info to title/description
						if desc, ok := branchSchema["description"].(string); ok {
							branchSchema["description"] = fmt.Sprintf("[%s] %s", contentType, desc)
						} else if title, ok := branchSchema["title"].(string); ok {
							branchSchema["title"] = fmt.Sprintf("[%s] %s", contentType, title)
						} else {
							branchSchema["title"] = fmt.Sprintf("Schema for %s", contentType)
						}
						oneOfDraft7 = append(oneOfDraft7, branchSchema)
					}
				}
				if len(oneOfDraft7) > 0 {
					// The property for the body will have a oneOf structure
					properties[arg.Name] = map[string]interface{}{
						"oneOf": oneOfDraft7,
					}
					// Use continue here to skip the common property handling below
					// as the property was already added with the oneOf structure.
					goto handled_arg_schema // Jump to the label after property handling
				} else {
					continue // Skip this body arg if no valid content type schemas
				}
			}
		} else {
			// For Path, Query, Header parameters, use Arg.Schema directly
			if arg.Schema == nil {
				continue // Skip arg if no schema
			}
			propSchemaMap, err = schemaToDraft7Map(arg.Schema) // Assign to err
			if err != nil {
				return "", fmt.Errorf("failed to convert schema for arg '%s': %w", arg.Name, err)
			}
		}

		// Common property handling for non-body args or single content type body
		if _, exists := properties[arg.Name]; !exists { // Ensure we don't overwrite if body oneOf was handled
			if propSchemaMap == nil {
				continue // Skip if schema conversion resulted in nil map
			}
			properties[arg.Name] = propSchemaMap
		}

	handled_arg_schema: // Label for goto

		if arg.Required {
			requiredProperties = append(requiredProperties, arg.Name)
		}
	}

	if len(properties) > 0 {
		rootSchema["properties"] = properties
	}

	if len(requiredProperties) > 0 {
		rootSchema["required"] = requiredProperties
	}

	schemaBytes, err := json.MarshalIndent(rootSchema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}

	return string(schemaBytes), nil
}

// schemaToDraft7Map converts a custom Schema struct to a JSON Schema Draft 7 map.
func schemaToDraft7Map(s *Schema) (map[string]interface{}, error) {
	if s == nil {
		return nil, nil
	}

	result := make(map[string]interface{})

	// --- Basic Metadata and Common Keywords ---
	if s.Title != "" {
		result["title"] = s.Title
	}
	if s.Description != "" {
		result["description"] = s.Description
	}
	if s.Format != "" {
		result["format"] = s.Format
	}
	if s.Default != nil {
		result["default"] = s.Default
	}
	if s.Example != nil {
		result["examples"] = []interface{}{s.Example}
	}
	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}
	if s.ReadOnly {
		result["readOnly"] = true
	}
	if s.WriteOnly {
		result["writeOnly"] = true
	}

	// --- Handle Schema Composition Keywords First ---
	if len(s.OneOf) > 0 {
		oneOfSchemas := make([]map[string]interface{}, len(s.OneOf))
		for i, subSchema := range s.OneOf {
			subSchemaMap, err := schemaToDraft7Map(subSchema) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert oneOf sub-schema: %w", err)
			}
			if subSchemaMap == nil { // Check for nil result from recursive call
				return nil, fmt.Errorf("oneOf sub-schema at index %d resulted in a nil schema map", i)
			}
			oneOfSchemas[i] = subSchemaMap
		}
		result["oneOf"] = oneOfSchemas
	}

	if len(s.AnyOf) > 0 {
		anyOfSchemas := make([]map[string]interface{}, len(s.AnyOf))
		for i, subSchema := range s.AnyOf {
			subSchemaMap, err := schemaToDraft7Map(subSchema) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert anyOf sub-schema: %w", err)
			}
			if subSchemaMap == nil { // Check for nil result from recursive call
				return nil, fmt.Errorf("anyOf sub-schema at index %d resulted in a nil schema map", i)
			}
			anyOfSchemas[i] = subSchemaMap
		}
		result["anyOf"] = anyOfSchemas
	}

	if len(s.AllOf) > 0 {
		allOfSchemas := make([]map[string]interface{}, len(s.AllOf))
		for i, subSchema := range s.AllOf {
			subSchemaMap, err := schemaToDraft7Map(subSchema) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("failed to convert allOf sub-schema: %w", err)
			}
			if subSchemaMap == nil { // Check for nil result from recursive call
				return nil, fmt.Errorf("allOf sub-schema at index %d resulted in a nil schema map", i)
			}
			allOfSchemas[i] = subSchemaMap // Corrected assignment
		}
		result["allOf"] = allOfSchemas
	}

	if s.Not != nil {
		notSchemaMap, err := schemaToDraft7Map(s.Not) // Recursive call
		if err != nil {
			return nil, fmt.Errorf("failed to convert not sub-schema: %w", err)
		}
		if notSchemaMap == nil { // Check for nil result from recursive call
			return nil, fmt.Errorf("not sub-schema resulted in a nil schema map")
		}
		result["not"] = notSchemaMap
	}

	// --- Handle Type(s) and Type-Specific Validations ---
	// Apply these based on the data in the struct, which came from OpenAPI.
	// schemaToDraft7Map is responsible for generating the *map* representation.

	if len(s.Types) == 1 {
		result["type"] = s.Types[0]
	} else if len(s.Types) > 1 {
		result["type"] = s.Types
	}
	// If len(s.Types) == 0, omit 'type' (any type in JS)

	// Apply ALL type-specific validations found in the validation substructs.
	// schemaToDraft7Map translates the fields from the substructs directly into map entries.
	if s.String != nil {
		if s.String.MinLength > 0 {
			result["minLength"] = s.String.MinLength
		}
		if s.String.MaxLength != nil {
			result["maxLength"] = *s.String.MaxLength
		}
		if s.String.Pattern != "" {
			result["pattern"] = s.String.Pattern
		}
	}
	if s.Number != nil {
		if s.Number.Minimum != nil {
			if s.Number.ExclusiveMinimum {
				result["exclusiveMinimum"] = *s.Number.Minimum
			} else {
				result["minimum"] = *s.Number.Minimum
			}
		}
		if s.Number.Maximum != nil {
			if s.Number.ExclusiveMaximum {
				result["exclusiveMaximum"] = *s.Number.Maximum
			} else {
				result["maximum"] = *s.Number.Maximum
			}
		}
		if s.Number.MultipleOf != nil {
			result["multipleOf"] = *s.Number.MultipleOf
		}
	}
	if s.Array != nil {
		if s.Array.Items != nil {
			itemsSchemaMap, err := schemaToDraft7Map(s.Array.Items) // Recursive call for items
			if err != nil {
				return nil, fmt.Errorf("failed to convert array items schema: %w", err)
			}
			if itemsSchemaMap != nil {
				result["items"] = itemsSchemaMap
			}
		}
		if s.Array.MinItems > 0 {
			result["minItems"] = s.Array.MinItems
		}
		if s.Array.MaxItems != nil {
			result["maxItems"] = *s.Array.MaxItems
		}
		if s.Array.UniqueItems {
			result["uniqueItems"] = true
		}
	}
	if s.Object != nil {
		if len(s.Object.Properties) > 0 {
			propertiesMap := make(map[string]interface{})
			for propName, propSchema := range s.Object.Properties {
				propSchemaMap, err := schemaToDraft7Map(propSchema) // Recursive call for properties
				if err != nil {
					return nil, fmt.Errorf("failed to convert property '%s': %w", propName, err)
				}
				if propSchemaMap != nil {
					propertiesMap[propName] = propSchemaMap
				}
			}
			if len(propertiesMap) > 0 {
				result["properties"] = propertiesMap
			}
		}
		if len(s.Object.Required) > 0 {
			result["required"] = s.Object.Required
		}
		if s.Object.MinProperties > 0 {
			result["minProperties"] = s.Object.MinProperties
		}
		if s.Object.MaxProperties != nil {
			result["maxProperties"] = *s.Object.MaxProperties
		}

		// Handle additionalProperties mapping
		if s.Object.DisallowAdditionalProperties {
			result["additionalProperties"] = false
		} else if s.Object.AdditionalProperties != nil {
			// If AdditionalProperties schema is provided in our struct, convert it recursively
			addPropSchemaMap, err := schemaToDraft7Map(s.Object.AdditionalProperties)
			if err != nil {
				return nil, fmt.Errorf("failed to convert additionalProperties schema: %w", err)
			}
			if addPropSchemaMap != nil {
				result["additionalProperties"] = addPropSchemaMap
			} else {
				// If additionalProperties was specified in OpenAPI (non-nil *Schema)
				// but its conversion resulted in a nil map, it might represent `{}`
				// or an invalid schema. Mapping to {} is equivalent to true.
				// Let's map a non-nil *Schema resolving to nil map here to {}.
				result["additionalProperties"] = map[string]interface{}{} // Represents {} schema (allow any)
			}
		}
	}

	return result, nil
}
