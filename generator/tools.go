package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/lyeslabs/mcpgen/converter"
)

// GenerateToolFiles generates individual tool files while preserving existing handler implementations
func (g *Generator) GenerateToolFiles(config *converter.MCPConfig) error {
	toolTemplateContent, err := templatesFS.ReadFile("templates/tool.templ")
	if err != nil {
		return fmt.Errorf("failed to read tool template file: %w", err)
	}

	tmpl, err := template.New("tool.templ").Parse(string(toolTemplateContent))
	if err != nil {
		return fmt.Errorf("failed to parse tool template: %w", err)
	}

	for _, tool := range config.Tools {
		data := struct {
			ToolTemplateData
			URL     string
			Method  string
			Headers []converter.Header
		}{
			ToolTemplateData: ToolTemplateData{
				ToolNameOriginal:      tool.Name,
				ToolNameGo:            tool.Name,
				ToolHandlerName:       tool.Name + "Handler",
				ToolDescription:       tool.Description,
				RawInputSchema:        tool.RawInputSchema,
				ResponseTemplate:      tool.ResponseTemplate.PrependBody,
				InputSchemaConst:      fmt.Sprintf("%sInputSchema", tool.Name),
				ResponseTemplateConst: fmt.Sprintf("%sResponseTemplate", tool.Name),
			},
			URL:     tool.RequestTemplate.URL,
			Method:  tool.RequestTemplate.Method,
			Headers: tool.RequestTemplate.Headers,
		}

		outputFileName := tool.Name + ".go"
		outputFilePath := filepath.Join(g.outputDir, outputFileName)

		// Check if file already exists and extract handler implementation if it does
		existingImplementation := ""
		existingImports := []string{}

		if _, err := os.Stat(outputFilePath); err == nil {
			existingContent, err := os.ReadFile(outputFilePath)
			if err == nil {
				existingImplementation = extractHandlerImplementation(string(existingContent), data.ToolHandlerName)
				existingImports = extractImports(string(existingContent))
			}
		}

		// Generate code for this tool
		var toolBuf bytes.Buffer

		// Write package declaration
		fmt.Fprintf(&toolBuf, "package %s\n\n", g.PackageName)

		// Merge imports
		requiredImports := []string{
			"context",
			"fmt",
			"github.com/mark3labs/mcp-go/mcp",
		}

		mergedImports := mergeImports(requiredImports, existingImports)

		if len(mergedImports) > 0 {
			fmt.Fprintf(&toolBuf, "import (\n")
			for _, imp := range mergedImports {
				fmt.Fprintf(&toolBuf, "\t\"%s\"\n", imp)
			}
			fmt.Fprintf(&toolBuf, ")\n\n")
		}

		// Execute template to get the boilerplate
		if err := tmpl.Execute(&toolBuf, data); err != nil {
			return fmt.Errorf("failed to render template for tool %s: %w", tool.Name, err)
		}

		// If we have an existing implementation, replace the default one
		if existingImplementation != "" {
			toolContent := toolBuf.String()
			toolContent = replaceHandlerImplementation(toolContent, data.ToolHandlerName, existingImplementation)
			toolBuf.Reset()
			toolBuf.WriteString(toolContent)
		}

		// Format the generated code
		formattedCode, err := format.Source(toolBuf.Bytes())
		if err != nil {
			return fmt.Errorf("failed to format generated code for %s: %w", outputFileName, err)
		}

		if err := os.MkdirAll(g.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(outputFilePath, formattedCode, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputFileName, err)
		}
	}

	return nil
}

// Import handling functions
func extractImports(fileContent string) []string {
	var imports []string

	// Find the import block
	importStart := strings.Index(fileContent, "import (")
	if importStart == -1 {
		// Try single-line imports
		re := regexp.MustCompile(`import\s+"([^"]+)"`)
		matches := re.FindAllStringSubmatch(fileContent, -1)
		for _, match := range matches {
			if len(match) > 1 {
				imports = append(imports, match[1])
			}
		}
		return imports
	}

	// Find the closing parenthesis
	importEnd := strings.Index(fileContent[importStart:], ")")
	if importEnd == -1 {
		return imports
	}

	importBlock := fileContent[importStart+8 : importStart+importEnd]

	// Extract each import
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(importBlock, -1)

	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}

	return imports
}

func mergeImports(required, existing []string) []string {
	uniqueImports := make(map[string]bool)

	// Add all required imports
	for _, imp := range required {
		uniqueImports[imp] = true
	}

	// Add existing imports that aren't already included
	for _, imp := range existing {
		uniqueImports[imp] = true
	}

	// Convert back to slice
	var result []string
	for imp := range uniqueImports {
		result = append(result, imp)
	}

	// Sort for consistent output
	sort.Strings(result)

	return result
}

func extractHandlerImplementation(fileContent, handlerName string) string {
	// Use regex to allow for flexible spacing
	pattern := fmt.Sprintf(`func\s+%s\s*\(\s*ctx\s+context\.Context\s*,\s*request\s+mcp\.CallToolRequest\s*\)\s*\(\s*\*mcp\.CallToolResult\s*,\s*error\s*\)\s*{`,
		regexp.QuoteMeta(handlerName))

	re := regexp.MustCompile(pattern)
	loc := re.FindStringIndex(fileContent)

	if loc == nil {
		log.Printf("Could not find handler function signature for %s", handlerName)
		return "" // Function not found
	}

	// Find the opening brace position (end of the regex match)
	startIdx := loc[1]

	// Track braces to find the matching closing brace
	braceCount := 1
	endIdx := startIdx

	for endIdx < len(fileContent) && braceCount > 0 {
		endIdx++
		if endIdx >= len(fileContent) {
			break
		}

		if fileContent[endIdx] == '{' {
			braceCount++
		} else if fileContent[endIdx] == '}' {
			braceCount--
		}
	}

	if endIdx >= len(fileContent) {
		log.Printf("Could not find matching closing brace for %s", handlerName)
		return "" // Couldn't find matching brace
	}

	// Extract everything between the opening and closing braces
	implementation := fileContent[startIdx:endIdx]

	return implementation
}

func replaceHandlerImplementation(fileContent, handlerName, implementation string) string {
	// Use a regex pattern to find the handler function
	pattern := fmt.Sprintf(`func\s+%s\s*\(\s*ctx\s+context\.Context\s*,\s*request\s+mcp\.CallToolRequest\s*\)\s*\(\s*\*mcp\.CallToolResult\s*,\s*error\s*\)\s*{`,
		regexp.QuoteMeta(handlerName))

	re := regexp.MustCompile(pattern)
	loc := re.FindStringIndex(fileContent)

	if loc == nil {
		log.Printf("Could not find handler function signature for: %s", handlerName)
		return fileContent
	}

	// Get the position after the opening brace
	startIdx := loc[1]

	// Track braces to find the matching closing brace
	braceCount := 1
	endIdx := startIdx

	for endIdx < len(fileContent) && braceCount > 0 {
		endIdx++
		if endIdx >= len(fileContent) {
			break
		}

		if fileContent[endIdx] == '{' {
			braceCount++
		} else if fileContent[endIdx] == '}' {
			braceCount--
		}
	}

	if endIdx >= len(fileContent) {
		log.Printf("Could not find matching closing brace for: %s", handlerName)
		return fileContent
	}

	// Create the new function body
	// Keep the original function declaration and closing brace
	newContent := fileContent[:startIdx] +
		"\n\t" + strings.TrimSpace(implementation) +
		"\n}" // Make sure we add the closing brace

	// Add everything after the original function
	if endIdx+1 < len(fileContent) {
		newContent += fileContent[endIdx+1:]
	}

	// Check if the last closing brace is already in the implementation
	if strings.HasSuffix(strings.TrimSpace(implementation), "}") {
		// If we have a closing brace in both the implementation and what we added,
		// we need to remove one of them
		newContent = fileContent[:startIdx] +
			"\n\t" + strings.TrimSpace(implementation) +
			fileContent[endIdx+1:]
	}

	return newContent
}
