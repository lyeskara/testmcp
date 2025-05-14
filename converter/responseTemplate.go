package converter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// SchemaInfo contains schema metadata used during schema traversal
type SchemaInfo struct {
	Path        string
	Schema      *openapi3.Schema
	ContentType string
	Indent      int
}

// createResponseTemplate creates a formatted template from an OpenAPI operation
func (c *Converter) createResponseTemplate(operation *openapi3.Operation) (*ResponseTemplate, error) {
	var b strings.Builder

	// Add header
	b.WriteString("# API Response Information\n\n")
	b.WriteString("Below is the response from an API call. To help you understand the data, I've provided:\n\n")
	b.WriteString("1. A detailed description of all fields in the response structure\n")
	b.WriteString("2. The complete API response\n\n")

	// Process success responses (2xx)
	if successSchemas := c.findSuccessSchemas(operation); len(successSchemas) > 0 {
		b.WriteString("## Success Response Structure\n\n")

		// Handle multiple success schemas
		for i, successSchema := range successSchemas {
			if len(successSchemas) > 1 {
				b.WriteString(fmt.Sprintf("### Success Model %d\n\n", i+1))
			}

			b.WriteString(fmt.Sprintf("> Content-Type: %s\n\n", successSchema.ContentType))
			c.writeSchemaDoc(&b, successSchema)
			b.WriteString("\n")
		}
	}

	// Process error responses (4xx, 5xx)
	if errorSchemas := c.findErrorSchemas(operation); len(errorSchemas) > 0 {
		b.WriteString("## Error Response Structure\n\n")

		// Handle multiple error schemas
		for i, errorSchema := range errorSchemas {
			if len(errorSchemas) > 1 {
				b.WriteString(fmt.Sprintf("### Error Model %d\n\n", i+1))
			}

			// Add description if available
			if errorSchema.Schema.Description != "" {
				b.WriteString(fmt.Sprintf("> Description: %s\n\n", errorSchema.Schema.Description))
			}

			b.WriteString(fmt.Sprintf("> Content-Type: %s\n\n", errorSchema.ContentType))
			c.writeSchemaDoc(&b, errorSchema)
			b.WriteString("\n")
		}
	}

	// Add the original response section header
	b.WriteString("## Original Response\n\n")

	return &ResponseTemplate{PrependBody: b.String()}, nil
}

// findSuccessSchemas extracts all success response schemas from an operation
func (c *Converter) findSuccessSchemas(operation *openapi3.Operation) []*SchemaInfo {
	if operation.Responses == nil {
		return nil
	}

	var successSchemas []*SchemaInfo
	uniqueSchemas := make(map[string]bool)

	// Find all 2xx responses with content
	for code, responseRef := range operation.Responses.Map() {
		if !strings.HasPrefix(code, "2") || responseRef == nil || responseRef.Value == nil {
			continue
		}

		for contentType, mediaType := range responseRef.Value.Content {
			if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
				continue
			}

			// Create a simple hash to identify duplicate schemas
			schemaHash := c.getSchemaHash(mediaType.Schema.Value)
			hashKey := contentType + ":" + schemaHash

			// Only add unique schemas
			if !uniqueSchemas[hashKey] {
				uniqueSchemas[hashKey] = true

				successSchemas = append(successSchemas, &SchemaInfo{
					Schema:      mediaType.Schema.Value,
					ContentType: contentType,
				})
			}
		}
	}

	return successSchemas
}

// findErrorSchemas extracts the error response schemas from an operation
func (c *Converter) findErrorSchemas(operation *openapi3.Operation) []*SchemaInfo {
	if operation.Responses == nil {
		return nil
	}

	var errorSchemas []*SchemaInfo
	uniqueSchemas := make(map[string]bool)

	// Find all 4xx and 5xx responses with content
	for code, responseRef := range operation.Responses.Map() {
		if !(strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5")) ||
			responseRef == nil || responseRef.Value == nil {
			continue
		}

		for contentType, mediaType := range responseRef.Value.Content {
			if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
				continue
			}

			// Create a simple hash to identify duplicate schemas
			schemaHash := c.getSchemaHash(mediaType.Schema.Value)
			hashKey := contentType + ":" + schemaHash

			// Only add unique schemas
			if !uniqueSchemas[hashKey] {
				uniqueSchemas[hashKey] = true

				errorSchemas = append(errorSchemas, &SchemaInfo{
					Schema:      mediaType.Schema.Value,
					ContentType: contentType,
				})
			}
		}
	}

	return errorSchemas
}

// getSchemaHash creates a simple signature to identify similar schemas
func (c *Converter) getSchemaHash(schema *openapi3.Schema) string {
	if schema == nil {
		return "nil"
	}

	var props []string
	for prop := range schema.Properties {
		props = append(props, prop)
	}
	sort.Strings(props)

	return strings.Join(props, ",")
}

// writeSchemaDoc writes documentation for a schema to the builder
func (c *Converter) writeSchemaDoc(b *strings.Builder, info *SchemaInfo) {
	if info == nil || info.Schema == nil {
		return
	}

	schema := info.Schema

	// Handle different schema types
	switch {
	case isArray(schema):
		b.WriteString("- **items**: Array of items (Type: array)\n")

		// Process array items if they exist
		if schema.Items != nil && schema.Items.Value != nil {
			itemInfo := &SchemaInfo{
				Path:   "items",
				Schema: schema.Items.Value,
				Indent: 1,
			}
			c.writeSchemaProperties(b, itemInfo)
		}

	case isObject(schema):
		// Process object properties directly
		c.writeSchemaProperties(b, info)

	default:
		// Handle primitive types
		if schema.Type != nil {
			typeName := "unknown"
			if len(*schema.Type) > 0 {
				typeName = (*schema.Type)[0]
			}
			b.WriteString(fmt.Sprintf("- Data of type: %s\n", typeName))
		}
	}
}

// writeSchemaProperties writes documentation for schema properties
func (c *Converter) writeSchemaProperties(b *strings.Builder, info *SchemaInfo) {
	if info == nil || info.Schema == nil || !isObject(info.Schema) {
		return
	}

	// Get sorted property names for consistent output
	var propNames []string
	for propName := range info.Schema.Properties {
		propNames = append(propNames, propName)
	}
	sort.Strings(propNames)

	// Calculate indentation
	indent := strings.Repeat("  ", info.Indent)

	// Process each property
	for _, propName := range propNames {
		propRef := info.Schema.Properties[propName]
		if propRef == nil || propRef.Value == nil {
			continue
		}

		propSchema := propRef.Value

		// Create property path
		propPath := propName
		if info.Path != "" {
			propPath = info.Path + "." + propName
		}

		// Write property info
		b.WriteString(fmt.Sprintf("%s- **%s**: %s", indent, propPath, propSchema.Description))
		if propSchema.Type != nil {
			b.WriteString(fmt.Sprintf(" (Type: %s)", (*propSchema.Type)[0]))
		}
		b.WriteString("\n")

		// Process nested properties recursively with increased indentation
		switch {
		case isArray(propSchema):
			// Handle array property
			if propSchema.Items != nil && propSchema.Items.Value != nil {
				arrayItemInfo := &SchemaInfo{
					Path:   propPath + "[]",
					Schema: propSchema.Items.Value,
					Indent: info.Indent + 1,
				}

				// If array items are objects, process their properties
				if isObject(propSchema.Items.Value) {
					c.writeSchemaProperties(b, arrayItemInfo)
				} else if propSchema.Items.Value.Type != nil {
					// If array items are primitive types, just note the type
					b.WriteString(fmt.Sprintf("%s  - Items of type: %s\n",
						indent, (*propSchema.Items.Value.Type)[0]))
				}
			}

		case isObject(propSchema):
			// Handle object property - traverse its properties recursively
			c.writeSchemaProperties(b, &SchemaInfo{
				Path:   propPath,
				Schema: propSchema,
				Indent: info.Indent + 1,
			})
		}
	}
}

// isArray checks if a schema represents an array
func isArray(schema *openapi3.Schema) bool {
	return schema != nil && schema.Type != nil &&
		len(*schema.Type) > 0 && (*schema.Type)[0] == "array"
}

// isObject checks if a schema represents an object with properties
func isObject(schema *openapi3.Schema) bool {
	return schema != nil && schema.Type != nil &&
		len(*schema.Type) > 0 && (*schema.Type)[0] == "object" &&
		len(schema.Properties) > 0
}

// getDescription returns a description for an operation
func getDescription(operation *openapi3.Operation) string {
	if operation.Summary != "" {
		if operation.Description != "" {
			return fmt.Sprintf("%s - %s", operation.Summary, operation.Description)
		}
		return operation.Summary
	}
	return operation.Description
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
