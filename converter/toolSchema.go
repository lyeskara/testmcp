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

		propSchema, err := buildPropertySchema(arg)
		if err != nil {
			return "", err
		}
		if propSchema == nil {
			continue
		}

		properties[arg.Name] = propSchema

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

// buildPropertySchema builds the JSON Schema property for a given Arg.
// Returns nil if the property should be skipped.
func buildPropertySchema(arg Arg) (map[string]interface{}, error) {
	var propSchema map[string]interface{}
	var err error

	switch arg.Source {
	case "body":
		propSchema, err = buildBodySchema(arg)
	default:
		if arg.Schema == nil {
			return nil, nil
		}
		propSchema, err = schemaToDraft7Map(arg.Schema)
	}
	if err != nil || propSchema == nil {
		return propSchema, err
	}

	if arg.Description != "" {
		if _, hasDesc := propSchema["description"]; !hasDesc {
			propSchema["description"] = arg.Description
		} else if propSchema["description"] == "" {
			propSchema["description"] = arg.Description
		}
	}

	return propSchema, nil
}

// buildBodySchema handles the "body" source, including multiple content types.
func buildBodySchema(arg Arg) (map[string]interface{}, error) {
	if len(arg.ContentTypes) == 0 {
		return nil, nil
	}
	if len(arg.ContentTypes) == 1 {
		// Only one content type, use its schema directly
		for _, schema := range arg.ContentTypes {
			return schemaToDraft7Map(schema)
		}
	}

	// Multiple content types: use oneOf
	oneOfSchemas := []map[string]interface{}{}
	for contentType, schema := range arg.ContentTypes {
		branchSchema, err := schemaToDraft7Map(schema)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to convert body schema branch for content type '%s': %w",
				contentType, err,
			)
		}
		if branchSchema != nil {
			// Add content type info to title/description
			addContentTypeInfo(branchSchema, contentType)
			oneOfSchemas = append(oneOfSchemas, branchSchema)
		}
	}
	if len(oneOfSchemas) == 0 {
		return nil, nil
	}
	return map[string]interface{}{
		"oneOf": oneOfSchemas,
	}, nil
}

// addContentTypeInfo adds content type info to the schema's title or description.
func addContentTypeInfo(schema map[string]interface{}, contentType string) {
	if desc, ok := schema["description"].(string); ok {
		schema["description"] = fmt.Sprintf("[%s] %s", contentType, desc)
	} else if title, ok := schema["title"].(string); ok {
		schema["title"] = fmt.Sprintf("[%s] %s", contentType, title)
	} else {
		schema["title"] = fmt.Sprintf("Schema for %s", contentType)
	}
}

func schemaToDraft7Map(s *Schema) (map[string]interface{}, error) {
	if s == nil {
		return nil, nil
	}

	result := make(map[string]interface{})

	addBasicMetadata(result, s)
	addType(result, s)
	addStringValidation(result, s)
	addNumberValidation(result, s)
    
	if err := addCombinators(result, s); err != nil {
		return nil, err
	}
	if err := addArrayValidation(result, s); err != nil {
		return nil, err
	}
	if err := addObjectValidation(result, s); err != nil {
		return nil, err
	}

	return result, nil
}

func addBasicMetadata(result map[string]interface{}, s *Schema) {
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
}

func addCombinators(result map[string]interface{}, s *Schema) error {
	if len(s.OneOf) > 0 {
		oneOfSchemas, err := convertSubSchemas(s.OneOf)
		if err != nil {
			return fmt.Errorf("failed to convert oneOf: %w", err)
		}
		result["oneOf"] = oneOfSchemas
	}
	if len(s.AnyOf) > 0 {
		anyOfSchemas, err := convertSubSchemas(s.AnyOf)
		if err != nil {
			return fmt.Errorf("failed to convert anyOf: %w", err)
		}
		result["anyOf"] = anyOfSchemas
	}
	if len(s.AllOf) > 0 {
		allOfSchemas, err := convertSubSchemas(s.AllOf)
		if err != nil {
			return fmt.Errorf("failed to convert allOf: %w", err)
		}
		result["allOf"] = allOfSchemas
	}
	if s.Not != nil {
		notSchemaMap, err := schemaToDraft7Map(s.Not)
		if err != nil {
			return fmt.Errorf("failed to convert not sub-schema: %w", err)
		}
		if notSchemaMap == nil {
			return fmt.Errorf("not sub-schema resulted in a nil schema map")
		}
		result["not"] = notSchemaMap
	}
	return nil
}

func convertSubSchemas(subSchemas []*Schema) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, len(subSchemas))
	for i, subSchema := range subSchemas {
		subSchemaMap, err := schemaToDraft7Map(subSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to convert sub-schema at index %d: %w", i, err)
		}
		if subSchemaMap == nil {
			return nil, fmt.Errorf("sub-schema at index %d resulted in a nil schema map", i)
		}
		result[i] = subSchemaMap
	}
	return result, nil
}

func addType(result map[string]interface{}, s *Schema) {
	if len(s.Types) == 1 {
		result["type"] = s.Types[0]
	} else if len(s.Types) > 1 {
		result["type"] = s.Types
	}
}

func addStringValidation(result map[string]interface{}, s *Schema) {
	if s.String == nil {
		return
	}
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

func addNumberValidation(result map[string]interface{}, s *Schema) {
	if s.Number == nil {
		return
	}
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

func addArrayValidation(result map[string]interface{}, s *Schema) error {
	if s.Array == nil {
		return nil
	}
	if s.Array.Items != nil {
		itemsSchemaMap, err := schemaToDraft7Map(s.Array.Items)
		if err != nil {
			return fmt.Errorf("failed to convert array items schema: %w", err)
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
	return nil
}

func addObjectValidation(result map[string]interface{}, s *Schema) error {
	if s.Object == nil {
		return nil
	}
	if len(s.Object.Properties) > 0 {
		propertiesMap := make(map[string]interface{})
		for propName, propSchema := range s.Object.Properties {
			propSchemaMap, err := schemaToDraft7Map(propSchema)
			if err != nil {
				return fmt.Errorf("failed to convert property '%s': %w", propName, err)
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
		addPropSchemaMap, err := schemaToDraft7Map(s.Object.AdditionalProperties)
		if err != nil {
			return fmt.Errorf("failed to convert additionalProperties schema: %w", err)
		}
		if addPropSchemaMap != nil {
			result["additionalProperties"] = addPropSchemaMap
		} else {
			result["additionalProperties"] = map[string]interface{}{}
		}
	}
	return nil
}
