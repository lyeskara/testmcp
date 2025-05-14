package generator

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/lyeslabs/mcpgen/converter"
)

type Generator struct {
	specPath    string
	PackageName string
	outputDir   string
	converter   *converter.Converter
	spec        *openapi3.T
}

func NewGenerator(specPath string, validation bool, packageName string, outputDir string) (*Generator, error) {
	parser := converter.NewParser(validation)
	err := parser.ParseFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing OpenAPI specification: %v", err)
	}

	return &Generator{
		specPath:    specPath,
		converter:   converter.NewConverter(parser),
		spec:        parser.GetDocument(),
		outputDir:   outputDir,
		PackageName: packageName,
	}, nil
}
