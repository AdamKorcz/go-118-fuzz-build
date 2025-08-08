package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AdamKorcz/go-118-fuzz-build/input"
)

// ConvertDirectory reads all files in inputDir, converts them using ParseGoTestcase,
// and writes them to outputDir. Both directories must already exist.
func ConvertDirectory(inputDir, outputDir string) error {
	// Validate output directory
	info, err := os.Stat(outputDir)
	if err != nil {
		return fmt.Errorf("output directory error: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path %s is not a directory", outputDir)
	}

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // skip subdirectories
		}

		inputPath := filepath.Join(inputDir, entry.Name())
		inputData, err := os.ReadFile(inputPath) // modern replacement for ioutil.ReadFile
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file %s: %v\n", inputPath, err)
			continue
		}

		converted, err := input.ParseGoTestcase(string(inputData))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse test case %s: %v\n", inputPath, err)
			continue
		}

		outputPath := filepath.Join(outputDir, entry.Name())
		err = os.WriteFile(outputPath, converted, 0644) // modern replacement for ioutil.WriteFile
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write output file %s: %v\n", outputPath, err)
			continue
		}

		fmt.Printf("Converted: %s -> %s\n", inputPath, outputPath)
	}

	return nil
}