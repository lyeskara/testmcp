package generator

import (
	"fmt"
	"strings"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
)

func (g *Generator) GenerateHTTPClient(includes []string) error {
	// Determine what to generate
	var generateTypes, generateClient bool
	for _, inc := range includes {
		switch strings.ToLower(inc) {
		case "types":
			generateTypes = true
		case "httpclient":
			generateClient = true
		}
	}

	if !generateTypes && !generateClient {
		return fmt.Errorf("no valid includes specified (must include 'types', 'httpclient', or both)")
	}

	// Prepare codegen configuration
	cfg := codegen.Configuration{
		PackageName: g.PackageName,
		Generate: codegen.GenerateOptions{
			Models: generateTypes,
			Client: generateClient,
		},
	}

	// Generate the code
	code, err := codegen.Generate(g.spec, cfg)
	if err != nil {
		return fmt.Errorf("code generation failed: %v", err)
	}

	// Write to file
	if err := writeFileContent(g.outputDir, "HttpClient.go", func() ([]byte, error) {
		return []byte(code), nil
	}); err != nil {
		return fmt.Errorf("failed to write generated code to file: %v", err)
	}

	return nil
}
