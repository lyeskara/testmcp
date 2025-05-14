package converter

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// ConvertRequestBody converts an OpenAPI request body to our Arg structures
func (c *Converter) convertRequestBody(requestBodyRef *openapi3.RequestBodyRef) (*Arg, error) {
	if requestBodyRef == nil || requestBodyRef.Value == nil {
		return nil, nil
	}

	requestBody := requestBodyRef.Value
	Arg := Arg{
		Name:         "body",
		Source:       "body",
		Description:  requestBody.Description,
		Required:     requestBody.Required,
		ContentTypes: make(map[string]*Schema),
	}

	// Process each content type
	validContent := false
	for contentType, mediaType := range requestBody.Content {
		if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
			continue
		}

		schema, err := c.applySchema(mediaType.Schema.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert schema for content type %s: %w", contentType, err)
		}

		if schema != nil {
			Arg.ContentTypes[contentType] = schema
			validContent = true
		}
	}

	if validContent {
		return &Arg, nil
	}

	return nil, nil
}

// ConvertParameters converts OpenAPI parameters to our Arg structures
func (c *Converter) convertParameters(parameters openapi3.Parameters) ([]Arg, error) {
	args := []Arg{}

	for i, paramRef := range parameters {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}

		param := paramRef.Value

		// Skip invalid parameters
		if param.Schema == nil || param.Schema.Value == nil {
			continue
		}

		// Convert the schema using our new function
		schema, err := c.applySchema(param.Schema.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert schema for parameter %s (index %d): %w",
				param.Name, i, err)
		}

		// Create an arg for this parameter
		arg := Arg{
			Name:        param.Name,
			Description: param.Description,
			Source:      param.In,
			Required:    param.Required,
			Schema:      schema,
			Deprecated:  param.Deprecated,
		}

		args = append(args, arg)
	}

	return args, nil
}

// applySchemaMetadata applies basic schema metadata to create a Schema
func (c *Converter) applySchema(schema *openapi3.Schema) (*Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("cannot apply metadata to nil schema")
	}

	// Create a new Schema
	result := &Schema{
		Title:       schema.Title,
		Description: schema.Description,
		Format:      schema.Format,
		Enum:        schema.Enum,
		Default:     schema.Default,
		Example:     schema.Example,
		ReadOnly:    schema.ReadOnly,
		WriteOnly:   schema.WriteOnly,
	}

	// Handle types, including nullable
	if schema.Type != nil {
		result.Types = *schema.Type
	} else {
		result.Types = []string{} // Empty type array
	}

	if schema.Nullable {
		isNullableAlreadyPresent := false
		if result.Types != nil {
			for _, t := range result.Types {
				if t == "null" {
					isNullableAlreadyPresent = true
					break
				}
			}
		} else {
			// If no types are set, default to string
			result.Types = append(result.Types, "string")
		}
		if !isNullableAlreadyPresent {
			result.Types = append(result.Types, "null")
		}
	}

	// Apply type-specific validations
	var err error

	if hasStringType(schema) {
		result.String = c.createStringValidation(schema)
	}

	if hasNumericType(schema) {
		result.Number = c.createNumberValidation(schema)
	}

	if hasArrayType(schema) {
		result.Array, err = c.createArrayValidation(schema)
		if err != nil {
			return nil, fmt.Errorf("error creating array validation: %w", err)
		}
	}

	if hasObjectType(schema) {
		result.Object, err = c.createObjectValidation(schema)
		if err != nil {
			return nil, fmt.Errorf("error creating object validation: %w", err)
		}
	}

	// Handle OneOf
	if len(schema.OneOf) > 0 {
		result.OneOf = make([]*Schema, len(schema.OneOf))
		for i, subSchemaRef := range schema.OneOf {
			if subSchemaRef == nil || subSchemaRef.Value == nil {
				return nil, fmt.Errorf("oneOf contains a nil schema reference or value at index %d", i)
			}
			subSchema, err := c.applySchema(subSchemaRef.Value) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("error processing oneOf sub-schema at index %d: %w", i, err)
			}
			if subSchema != nil {
				result.OneOf[i] = subSchema
			} else {
				// Handle case where recursive call returned nil schema but no error?
				// This might indicate an issue in the source or applySchema logic.
				// For now, return error as a nil schema here is likely unexpected.
				return nil, fmt.Errorf("oneOf sub-schema at index %d resulted in a nil schema", i)
			}
		}
	}

	// Handle AnyOf
	if len(schema.AnyOf) > 0 {
		result.AnyOf = make([]*Schema, len(schema.AnyOf))
		for i, subSchemaRef := range schema.AnyOf {
			if subSchemaRef == nil || subSchemaRef.Value == nil {
				return nil, fmt.Errorf("anyOf contains a nil schema reference or value at index %d", i)
			}
			subSchema, err := c.applySchema(subSchemaRef.Value) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("error processing anyOf sub-schema at index %d: %w", i, err)
			}
			if subSchema != nil {
				result.AnyOf[i] = subSchema
			} else {
				return nil, fmt.Errorf("anyOf sub-schema at index %d resulted in a nil schema", i)
			}
		}
	}

	// Handle AllOf
	if len(schema.AllOf) > 0 {
		result.AllOf = make([]*Schema, len(schema.AllOf))
		for i, subSchemaRef := range schema.AllOf {
			if subSchemaRef == nil || subSchemaRef.Value == nil {
				return nil, fmt.Errorf("allOf contains a nil schema reference or value at index %d", i)
			}
			subSchema, err := c.applySchema(subSchemaRef.Value) // Recursive call
			if err != nil {
				return nil, fmt.Errorf("error processing allOf sub-schema at index %d: %w", i, err)
			}
			if subSchema != nil {
				result.AllOf[i] = subSchema
			} else {
				return nil, fmt.Errorf("allOf sub-schema at index %d resulted in a nil schema", i)
			}
		}
	}

	// Handle Not
	if schema.Not != nil && schema.Not.Value != nil {
		notSchema, err := c.applySchema(schema.Not.Value) // Recursive call
		if err != nil {
			return nil, fmt.Errorf("error processing not sub-schema: %w", err)
		}
		if notSchema != nil {
			result.Not = notSchema
		} else {
			return nil, fmt.Errorf("not sub-schema resulted in a nil schema")
		}
	}

	return result, nil
}

// createStringValidation creates string-specific validations
func (c *Converter) createStringValidation(schema *openapi3.Schema) *StringValidation {
	if schema == nil {
		return nil
	}
	return &StringValidation{
		MinLength: schema.MinLength,
		MaxLength: schema.MaxLength,
		Pattern:   schema.Pattern,
	}
}

// createNumberValidation creates number-specific validations
func (c *Converter) createNumberValidation(schema *openapi3.Schema) *NumberValidation {
	if schema == nil {
		return nil
	}
	return &NumberValidation{
		Minimum:          schema.Min,
		Maximum:          schema.Max,
		MultipleOf:       schema.MultipleOf,
		ExclusiveMinimum: schema.ExclusiveMin, // Maps getkin bool to our bool
		ExclusiveMaximum: schema.ExclusiveMax, // Maps getkin bool to our bool
	}
}

// createArrayValidation creates array-specific validations
func (c *Converter) createArrayValidation(schema *openapi3.Schema) (*ArrayValidation, error) {
	if schema == nil {
		return nil, nil // Return nil validation, no error
	}
	result := &ArrayValidation{
		MinItems:    schema.MinItems,
		MaxItems:    schema.MaxItems,
		UniqueItems: schema.UniqueItems,
	}

	if schema.Items != nil && schema.Items.Value != nil {
		itemsSchema, err := c.applySchema(schema.Items.Value)
		if err != nil {
			return nil, fmt.Errorf("error processing array items schema: %w", err)
		}
		result.Items = itemsSchema
	}

	return result, nil
}

// createObjectValidation creates object-specific validations
func (c *Converter) createObjectValidation(schema *openapi3.Schema) (*ObjectValidation, error) {
	// This check prevents panic if createObjectValidation is called with a nil schema
	if schema == nil {
		return nil, nil
	}

	result := &ObjectValidation{
		Required:      schema.Required,
		MinProperties: schema.MinProps,
		MaxProperties: schema.MaxProps,
	}

	if len(schema.Properties) > 0 {
		result.Properties = make(map[string]*Schema)
		for propName, propSchemaRef := range schema.Properties {
			// Check the SchemaRef pointer itself first
			if propSchemaRef != nil {
				// --- START OF CODE CHANGE SNIPPET ---
				// Now, check the *openapi3.Schema pointer within the SchemaRef's Value field
				if propSchemaRef.Value != nil { // <-- ADDED THIS CHECK
					propSchema, err := c.applySchema(propSchemaRef.Value) // Pass the non-nil Value
					if err != nil {
						return nil, fmt.Errorf("error processing property '%s': %w", propName, err)
					}
					if propSchema != nil { // Only add if conversion was successful
						result.Properties[propName] = propSchema
					}
				} else {
					// propSchemaRef was non-nil, but its Value was nil.
					// This can represent a property defined with `{}` in the OpenAPI spec.
					// Map this to an empty schema in our struct.
					result.Properties[propName] = &Schema{} // Map {}
					// Optional: Log a warning if this case is unexpected
					// fmt.Printf("Warning: Property '%s' has a non-nil SchemaRef but nil Value. Mapping to empty schema.\n", propName)
				}
				// --- END OF CODE CHANGE SNIPPET ---
			} else {
				// Optional: Log a warning for a nil property schema reference if this shouldn't happen
				// fmt.Printf("Warning: Property '%s' has a nil schema reference in the OpenAPI spec.\n", propName)
			}
		}
		if len(result.Properties) == 0 {
			result.Properties = nil
		}
	}

	// --- Handle additionalProperties ---
	// schema.AdditionalProperties is a VALUE type, so it's never nil.
	// Check its pointer fields (.Has and .Schema) for nil.

	if schema.AdditionalProperties.Has != nil {
		// Case 1: additionalProperties is explicitly true or false
		if !*schema.AdditionalProperties.Has { // Safely dereference Has pointer
			result.DisallowAdditionalProperties = true
		} else { // *schema.AdditionalProperties.Has is true
			result.AdditionalProperties = &Schema{} // Represents allowing any additional properties ({})
		}
	} else if schema.AdditionalProperties.Schema != nil {
		// Case 2: additionalProperties is a schema object (or meant to be)
		// --- START OF CODE CHANGE SNIPPET ---
		// Check if the resolved Schema pointer within the SchemaRef is non-nil
		if schema.AdditionalProperties.Schema.Value != nil { // <-- ADDED THIS CHECK
			addPropSchema, err := c.applySchema(schema.AdditionalProperties.Schema.Value) // Pass non-nil Value
			if err != nil {
				return nil, fmt.Errorf("error processing additionalProperties schema: %w", err)
			}
			if addPropSchema != nil {
				result.AdditionalProperties = addPropSchema
			} else {
				// SchemaRef.Value was non-nil, but applySchema returned nil. Map to {}.
				result.AdditionalProperties = &Schema{}
			}
		} else {
			// schema.AdditionalProperties.Schema is non-nil, but its Value is nil.
			// This corresponds to additionalProperties: {}. Map to {}.
			result.AdditionalProperties = &Schema{}
			// Optional: Log a warning if this case is unexpected
			// fmt.Printf("Warning: AdditionalProperties SchemaRef has a nil Value. Mapping to empty schema {}.\n")
		}
		// --- END OF CODE CHANGE SNIPPET ---
	}
	// If both .Has and .Schema pointers are nil, additionalProperties was omitted (default true).
	// Our struct defaults (nil AdditionalProperties, false DisallowAdditionalProperties) handle this.

	return result, nil
}

// Type checking helpers - now with nil checking
func hasStringType(schema *openapi3.Schema) bool {
	return schema != nil && schema.Type != nil && contains(*schema.Type, "string")
}

func hasNumericType(schema *openapi3.Schema) bool {
	return schema != nil && schema.Type != nil &&
		(contains(*schema.Type, "number") || contains(*schema.Type, "integer"))
}

func hasArrayType(schema *openapi3.Schema) bool {
	return schema != nil && schema.Type != nil && contains(*schema.Type, "array")
}

func hasObjectType(schema *openapi3.Schema) bool {
	return schema != nil && schema.Type != nil && contains(*schema.Type, "object")
}
