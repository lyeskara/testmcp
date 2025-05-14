package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// File utility functions
func writeFileContent(outputDir, fileName string, generateContent func() ([]byte, error)) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(outputDir, fileName)
	existingContent, readErr := os.ReadFile(filePath)

	newContent, err := generateContent()
	if err != nil {
		return err
	}

	// Compare content - handle nil existingContent if file didn't exist
	contentEqual := false
	if readErr == nil && existingContent != nil { // Only compare if file existed and was read
		contentEqual = bytes.Equal(existingContent, newContent)
	} else if readErr != nil && len(newContent) == 0 {
		contentEqual = true
	}

	if !contentEqual {
		if err := os.WriteFile(filePath, newContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fileName, err)
		}
	}

	return nil
}
