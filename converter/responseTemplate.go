package converter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func (c *Converter) createResponseTemplates(operation *openapi3.Operation) ([]ResponseTemplate, error) {
	if operation == nil || operation.Responses == nil {
		return nil, nil
	}

	var templates []ResponseTemplate

	// Process responses in sorted order of status code for consistent suffix assignment
	sortedCodes := []string{}
	for code := range operation.Responses.Map() {
		sortedCodes = append(sortedCodes, code)
	}
	// Atoi for numerical sort, then alphabetical for non-numerical (like "default")
	// This ensures 200 comes before 400, and different content types for the same code are grouped.
	sort.SliceStable(sortedCodes, func(i, j int) bool {
		codeI, errI := strconv.Atoi(sortedCodes[i])
		codeJ, errJ := strconv.Atoi(sortedCodes[j])
		if errI == nil && errJ == nil {
			return codeI < codeJ
		}
		if errI != nil && errJ != nil {
			return sortedCodes[i] < sortedCodes[j] // Alphabetical for non-numeric
		}
		return errI == nil // Numeric codes come before non-numeric ("default")
	})

	for _, code := range sortedCodes {
		responseRef := operation.Responses.Map()[code]

		if responseRef == nil || responseRef.Value == nil {
			continue
		}
		statusCode, _ := strconv.Atoi(code) // fallback to 0 if not int

		// Sort content types for consistency
		contentTypes := []string{}
		for ct := range responseRef.Value.Content {
			contentTypes = append(contentTypes, ct)
		}
		sort.Strings(contentTypes)

		for _, contentType := range contentTypes {
			mediaType := responseRef.Value.Content[contentType]

			if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
				continue
			}
			schema := mediaType.Schema.Value

			var b strings.Builder
			b.WriteString("# API Response Information\n\n")
			b.WriteString("Below is the response template for this API endpoint.\n\n")
			b.WriteString("The template shows a possible response, including its status code and content type, to help you understand and generate correct outputs.\n\n")
			b.WriteString(fmt.Sprintf("**Status Code:** %s\n\n", code))
			b.WriteString(fmt.Sprintf("**Content-Type:** %s\n\n", contentType))
			if responseRef.Value.Description != nil && *responseRef.Value.Description != "" {
				b.WriteString(fmt.Sprintf("> %s\n\n", *responseRef.Value.Description))
			}
			b.WriteString("## Response Structure\n\n")

			// Use the new function to write the schema documentation in Markdown
			c.writeSchemaMarkdown(&b, schema, 0, "")

			templates = append(templates, ResponseTemplate{
				PrependBody: b.String(),
				StatusCode:  statusCode,
				ContentType: contentType,
			})
		}
	}
	return assignSuffixes(templates), nil
}

// assignSuffixes adds a unique letter suffix (_A, _B, ...) to each response template.
// Assumes the input slice is already ordered correctly.
func assignSuffixes(responses []ResponseTemplate) []ResponseTemplate {
	for i := range responses {
		responses[i].Suffix = string('A' + i)
	}
	return responses
}

// writeSchemaMarkdown recursively writes the schema documentation in a readable Markdown format.
// It focuses on describing the structure and types clearly, handling objects, arrays,
// and combinators (oneOf, anyOf, allOf, not).
func (c *Converter) writeSchemaMarkdown(
	b *strings.Builder,
	schema *openapi3.Schema,
	indent int,
	fieldName string, // The name of the field/property this schema belongs to (e.g., "errors", "items"). Used for labeling.
) {
	if schema == nil {
		return
	}

	ind := strings.Repeat("  ", indent) // Markdown indentation

	// Determine primary type(s) and format
	typeStrs := []string{}
	if schema.Type != nil {
		typeStrs = *schema.Type
	}
	if schema.Format != "" {
		typeStrs = append(typeStrs, schema.Format) // Add format to type string if present
	}
	typeDesc := strings.Join(typeStrs, ", ")
	if typeDesc == "" && (len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 || len(schema.AllOf) > 0 || schema.Not != nil) {
		typeDesc = "Combinator" // Indicate it's a combination schema
	} else if typeDesc == "" {
		typeDesc = "unknown"
	}

	// --- Print the introductory line for this schema/field ---
	// This function is called for the root schema, object properties, and array items.

	description := schema.Description
	if fieldName != "" {
		// It's a named field (object property or array 'Items')
		if description == "" {
			b.WriteString(fmt.Sprintf("%s- **%s** (Type: %s):\n", ind, fieldName, typeDesc)) // Add colon for nested structure
		} else {
			b.WriteString(fmt.Sprintf("%s- **%s**: %s (Type: %s):\n", ind, fieldName, description, typeDesc)) // Add colon
		}
	} else {
		// It's the root schema or a combinator option/part
		if description == "" {
			b.WriteString(fmt.Sprintf("%s- Structure (Type: %s):\n", ind, typeDesc)) // Add colon for nested structure
		} else {
			b.WriteString(fmt.Sprintf("%s- %s (Type: %s):\n", ind, description, typeDesc)) // Add colon
		}
	}

	// --- Print non-structural details (validation rules, examples, default etc.) ---
	// Add a slight indent for details under the main line
	c.writeSchemaDetails(b, schema, indent+1)

	// --- Handle nested structures (Objects, Arrays, Combinators) ---

	// Handle Object properties
	if isObject(schema) && len(schema.Properties) > 0 {
		// No need for a "Properties:" line, the nested bullets show properties
		for propName, propRef := range schema.Properties {
			if propRef != nil && propRef.Value != nil {
				c.writeSchemaMarkdown(b, propRef.Value, indent+1, propName) // Recurse for property
			}
		}
	}

	// Handle Arrays
	if isArray(schema) && schema.Items != nil && schema.Items.Value != nil {
		// Describe the items schema under the array bullet
		c.writeSchemaMarkdown(b, schema.Items.Value, indent+1, "Items") // Use "Items" as the field name for array elements
	}

	// Handle Combinators
	// Describe the relationship and then document each sub-schema.
	if len(schema.OneOf) > 0 {
		b.WriteString(fmt.Sprintf("%s  - **One Of the following structures**:\n", ind)) // Clear label for oneOf
		for i, sub := range schema.OneOf {
			// Label each option clearly
			optionLabel := fmt.Sprintf("Option %d", i+1)
			// Recursively document each option, treat it like a sub-structure
			c.writeSchemaMarkdown(b, sub.Value, indent+2, optionLabel) // Increase indent, use label
		}
	}
	if len(schema.AnyOf) > 0 {
		b.WriteString(fmt.Sprintf("%s  - **Any Of the following structures**:\n", ind)) // Clear label for anyOf
		for i, sub := range schema.AnyOf {
			optionLabel := fmt.Sprintf("Option %d", i+1)
			c.writeSchemaMarkdown(b, sub.Value, indent+2, optionLabel)
		}
	}
	if len(schema.AllOf) > 0 {
		b.WriteString(fmt.Sprintf("%s  - **Combines All Of the following structures**:\n", ind)) // Clear label for allOf
		for i, sub := range schema.AllOf {
			optionLabel := fmt.Sprintf("Part %d", i+1) // Use "Part" for allOf as it's composition
			c.writeSchemaMarkdown(b, sub.Value, indent+2, optionLabel)
		}
	}
	if schema.Not != nil && schema.Not.Value != nil {
		b.WriteString(fmt.Sprintf("%s  - **Not**: Cannot be the following structure:\n", ind)) // Clear label for not
		c.writeSchemaMarkdown(b, schema.Not.Value, indent+2, "Forbidden Structure")            // Describe what's not allowed
	}

	// Handle additional properties if they have a schema
	if isObject(schema) && schema.AdditionalProperties.Schema != nil && schema.AdditionalProperties.Schema.Value != nil {
		b.WriteString(fmt.Sprintf("%s  - **Additional Properties**:\n", ind))
		c.writeSchemaMarkdown(b, schema.AdditionalProperties.Schema.Value, indent+2, "property value") // Describe the value schema
	} else if isObject(schema) && schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		// Handle case where additional properties are allowed but have no specified schema
		b.WriteString(fmt.Sprintf("%s  - **Allows Additional Properties**\n", ind))
	}
}

// writeSchemaDetails adds validation rules, examples, and default values in Markdown
func (c *Converter) writeSchemaDetails(b *strings.Builder, schema *openapi3.Schema, indent int) {
	ind := strings.Repeat("  ", indent)

	details := []string{}

	// Helper to safely format examples and defaults for Go raw strings
	formatForGoRawString := func(value interface{}) string {
		// Convert to string representation
		str := fmt.Sprintf("%v", value)
		if bts, err := json.Marshal(value); err == nil {
			str = string(bts)
		}

		// For string values, remove the JSON quotes
		if schema.Type != nil && len(*schema.Type) > 0 && (*schema.Type)[0] == "string" {
			if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
				str = strings.Trim(str, `"`)
			}
		}

		// Replace backticks with single quotes for Markdown
		str = strings.ReplaceAll(str, "`", "'")
		return str
	}

	// String validations
	if schema.MinLength > 0 {
		details = append(details, fmt.Sprintf("Min Length: %d", schema.MinLength))
	}
	if schema.MaxLength != nil && *schema.MaxLength > 0 {
		details = append(details, fmt.Sprintf("Max Length: %d", *schema.MaxLength))
	}
	if schema.Pattern != "" {
		// Use single quotes instead of backticks for patterns
		details = append(details, fmt.Sprintf("Pattern: '%s'", strings.ReplaceAll(schema.Pattern, "`", "'")))
	}

	// [Keep all other validation rules the same...]

	// Default/Example handling - using our new safe formatter
	if schema.Default != nil {
		formattedDefault := formatForGoRawString(schema.Default)
		details = append(details, fmt.Sprintf("Default: '%s'", formattedDefault)) // Using single quotes
	}
	if schema.Example != nil {
		formattedExample := formatForGoRawString(schema.Example)
		details = append(details, fmt.Sprintf("Example: '%s'", formattedExample)) // Using single quotes
	}

	// Handle enums
	if len(schema.Enum) > 0 {
		enumStrings := make([]string, len(schema.Enum))
		for i, e := range schema.Enum {
			formattedEnum := formatForGoRawString(e)
			enumStrings[i] = fmt.Sprintf("'%s'", formattedEnum) // Using single quotes
		}
		details = append(details, fmt.Sprintf("Enum: [%s]", strings.Join(enumStrings, ", ")))
	}

	// Print details
	if len(details) > 0 {
		detailIndent := ind + "  "
		for _, detail := range details {
			b.WriteString(fmt.Sprintf("%s- %s\n", detailIndent, detail))
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
