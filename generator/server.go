package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"text/template"

	"github.com/lyeslabs/mcpgen/converter"
)

// GenerateServerFile creates a server.go file in the same package as the tools
func (g *Generator) GenerateServerFile(config *converter.MCPConfig) error {
	serverTemplateContent, err := templatesFS.ReadFile("templates/server.templ")
	if err != nil {
		return fmt.Errorf("failed to read server template file: %w", err)
	}

	tmpl, err := template.New("server.templ").Parse(string(serverTemplateContent))
	if err != nil {
		return fmt.Errorf("failed to parse server template: %w", err)
	}

	data := struct {
		PackageName string
		Tools       []ToolTemplateData
	}{
		PackageName: g.PackageName,
		Tools:       make([]ToolTemplateData, 0, len(config.Tools)),
	}

	for _, tool := range config.Tools {
		data.Tools = append(data.Tools, ToolTemplateData{
			ToolNameOriginal: tool.Name,
			ToolNameGo:       tool.Name,
			ToolHandlerName:  tool.Name + "Handler",
			ToolDescription:  tool.Description,
		})
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render server template: %w", err)
	}

	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format generated server.go: %w", err)
	}

	if err := writeFileContent(g.outputDir, "server.go", func() ([]byte, error) {
		return formattedCode, nil
	}); err != nil {
		return fmt.Errorf("failed to write server.go file: %w", err)
	}

	return nil
}
