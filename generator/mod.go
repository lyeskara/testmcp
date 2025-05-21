package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxSearchDepth = 50
	goModFile      = "go.mod"
)

// BuildImportPath finds the module root and builds the import path for mcptools
func BuildImportPath(outputDir string) (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find module info
	moduleName, moduleRoot, err := findModulePath(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to find module: %w", err)
	}

	// Get absolute path of mcptools directory
	mcptoolsPath := filepath.Join(cwd, outputDir, "mcptools")

	// Calculate relative path from module root to mcptools
	relPath, err := filepath.Rel(moduleRoot, mcptoolsPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Build the complete import path
	importPath := filepath.ToSlash(filepath.Join(moduleName, relPath))
	return importPath, nil
}

// findModulePath searches upward from startDir to find the go.mod file
func findModulePath(startDir string) (string, string, error) {
	currentDir := filepath.Clean(startDir)
	for depth := 0; depth < maxSearchDepth; depth++ {
		goModPath := filepath.Join(currentDir, goModFile)
		
		// Check if go.mod exists
		if _, err := os.Stat(goModPath); err == nil {
			moduleName, err := parseModuleName(goModPath)
			if err != nil {
				return "", "", err
			}
			return moduleName, currentDir, nil
		} else if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("error checking for go.mod: %w", err)
		}

		// Move up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached filesystem root
		}
		currentDir = parent
	}

	return "", "", fmt.Errorf("no go.mod found in %d levels from %s", 
		maxSearchDepth, startDir)
}

// parseModuleName extracts the module name from go.mod file
func parseModuleName(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", goModPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Extract module name
			modulePart := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			
			// Remove comments
			if i := strings.Index(modulePart, "//"); i != -1 {
				modulePart = strings.TrimSpace(modulePart[:i])
			}
			
			// Handle quotes
			if len(modulePart) > 0 && (modulePart[0] == '"' || modulePart[0] == '\'') {
				quote := modulePart[0]
				if end := strings.IndexByte(modulePart[1:], quote); end != -1 {
					return modulePart[1 : end+1], nil
				}
				return "", fmt.Errorf("unclosed quote in module declaration")
			}
			
			// Return first word (stops at space or comment)
			return strings.Fields(modulePart)[0], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning go.mod: %w", err)
	}

	return "", fmt.Errorf("module declaration not found in %s", goModPath)
}
