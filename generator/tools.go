package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
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
		capitalizedName := capitalizeFirstLetter(tool.Name)
		data := struct {
			ToolTemplateData
			URL     string
			Method  string
			Headers []converter.Header
		}{
			ToolTemplateData: ToolTemplateData{
				ToolNameOriginal:      capitalizedName,
				ToolNameGo:            capitalizedName,
				ToolHandlerName:       capitalizedName + "Handler",
				ToolDescription:       tool.Description,
				RawInputSchema:        tool.RawInputSchema,
				ResponseTemplate:      tool.Responses,
				InputSchemaConst:      fmt.Sprintf("%sInputSchema", tool.Name),
				ResponseTemplateConst: fmt.Sprintf("%sResponseTemplate", tool.Name),
			},
			URL:     tool.RequestTemplate.URL,
			Method:  tool.RequestTemplate.Method,
			Headers: tool.RequestTemplate.Headers,
		}

		outputFileName := capitalizedName + ".go"
		outputFilePath := filepath.Join(g.outputDir+"/mcptools", outputFileName)

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
		fmt.Fprintf(&toolBuf, "package mcptools\n\n")

		// Merge imports
		requiredImports := []string{
			"context",
			"fmt",
			"github.com/mark3labs/mcp-go/mcp",
		}

		if len(existingImports) > 0 {
			fmt.Fprintf(&toolBuf, "import (\n")
			for _, imp := range existingImports {
				fmt.Fprintf(&toolBuf, "\t%s\n", imp)
			}
			fmt.Fprintf(&toolBuf, ")\n\n")
		} else {
			fmt.Fprintf(&toolBuf, "import (\n")
			for _, imp := range requiredImports {
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

		err = writeFileContent(g.outputDir+"/mcptools", outputFileName, func() ([]byte, error) {
			return formattedCode, nil
		})

		if err != nil {
			return fmt.Errorf("failed to write %s: %w", outputFileName, err)
		}
	}

	return nil
}

func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func extractImports(fileContent string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", fileContent, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	var imports []string
	for _, imp := range f.Imports {
		importLine := ""
		if imp.Name != nil {
			importLine += imp.Name.Name + " "
		}
		importLine += imp.Path.Value // includes quotes
		imports = append(imports, importLine)
	}
	return imports
}

// Extracts the full function body (as string) for a handler with the given name and signature.
func extractHandlerImplementation(fileContent, handlerName string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", fileContent, parser.ParseComments)
	if err != nil {
		return ""
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != handlerName {
			continue
		}
		// Check signature: (ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
		if len(fn.Type.Params.List) != 2 || len(fn.Type.Results.List) != 2 {
			continue
		}
		param2 := fn.Type.Params.List[1]
		result1 := fn.Type.Results.List[0]

		if exprToString(param2.Type) != "mcp.CallToolRequest" {
			continue
		}
		if exprToString(result1.Type) != "*mcp.CallToolResult" {
			continue
		}

		// Print the function body as Go code
		if fn.Body != nil {
			// Get the body as a substring, including braces
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < end && end < len(fileContent) {
				body := fileContent[start : end+1]
				// Ensure trailing newline for clean formatting
				if !strings.HasSuffix(body, "\n") {
					body += "\n"
				}
				return body
			}
		}
	}
	return ""
}

func replaceHandlerImplementation(fileContent, handlerName, implementation string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", fileContent, parser.ParseComments)
	if err != nil {
		return fileContent
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != handlerName {
			continue
		}
		if fn.Body == nil {
			continue
		}
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		if start < end && end < len(fileContent) {
			var buf bytes.Buffer
			buf.WriteString(fileContent[:start])
			impl := implementation
			if !strings.HasSuffix(impl, "\n") {
				impl += "\n"
			}
			buf.WriteString(impl)
			buf.WriteString(fileContent[end+1:])
			return buf.String()
		}
	}
	return fileContent
}

func exprToString(expr ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), expr)
	return buf.String()
}
