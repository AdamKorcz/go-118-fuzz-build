// cmd/zip-gofuzz-corpus/main.go
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/AdamKorcz/go-118-fuzz-build/input"
)

func main() {
	var (
		dir        = flag.String("dir", "", "Directory containing stdlib Go fuzz corpus files (with 'go test fuzz v1' header).")
		file       = flag.String("file", "", "Path to a single stdlib Go fuzz corpus file to add.")
		fuzzerName = flag.String("fuzzer_name", "", "Base name of the fuzzer; output will be <fuzzer_name>_seed_corpus.zip in $OUT.")
		verbose    = flag.Bool("verbose", false, "Print a summary of added/skipped files.")
	)
	flag.Parse()

	// Validate flags
	if strings.TrimSpace(*fuzzerName) == "" {
		log.Fatalf("missing -fuzzer_name")
	}
	if (*dir == "" && *file == "") || (*dir != "" && *file != "") {
		log.Fatalf("you must specify exactly one of -dir or -file")
	}

	// Check $OUT early so users get a nice error before we do any work.
	outDir := os.Getenv("OUT")
	if outDir == "" {
		log.Fatalf("$OUT is not set (the zip is written there).")
	}
	if fi, err := os.Stat(outDir); err != nil || !fi.IsDir() {
		log.Fatalf("$OUT must point to an existing directory (got %q)", outDir)
	}

	// Normalize fuzzer name and build outputName (without .zip), e.g. "myFuzzer_seed_corpus"
	name := strings.TrimSpace(*fuzzerName)
	name = strings.TrimSuffix(name, ".zip")
	name = strings.TrimSuffix(name, "_seed_corpus")
	if name == "" {
		log.Fatalf("invalid -fuzzer_name")
	}
	outputName := name + "_seed_corpus" // ZipCorpusFromGoFuzzCases will write $OUT/<outputName>.zip

	finalZip := filepath.Join(outDir, outputName+".zip")

	// Run in chosen mode
	switch {
	case *dir != "":
		// Validate directory
		if fi, err := os.Stat(*dir); err != nil || !fi.IsDir() {
			log.Fatalf("-dir must point to an existing directory (got %q)", *dir)
		}
		if *verbose {
			fmt.Printf("Input dir : %s\n", *dir)
			fmt.Printf("Output zip: %s\n", finalZip)
		}
		if err := input.ZipCorpusFromGoFuzzCases(*dir, outputName, *verbose); err != nil {
			log.Fatalf("zip (dir) failed: %v", err)
		}
	case *file != "":
		// Validate file
		if fi, err := os.Stat(*file); err != nil || fi.IsDir() {
			log.Fatalf("-file must point to an existing file (got %q)", *file)
		}
		if *verbose {
			fmt.Printf("Input file: %s\n", *file)
			fmt.Printf("Output zip: %s\n", finalZip)
		}
		if err := zipSingleGoFuzzCase(*file, outputName, *verbose); err != nil {
			log.Fatalf("zip (single file) failed: %v", err)
		}
	}

	if *verbose {
		fmt.Println("Done.")
	}
}

// zipSingleGoFuzzCase stages a single stdlib-format fuzz case into a temporary
// directory and then calls input.ZipCorpusFromGoFuzzCases so that the exact same
// conversion code path is used as the directory case.
func zipSingleGoFuzzCase(inputFile, outputName string, verbose bool) error {
	tmpDir, err := os.MkdirTemp("", "single-gofuzz-case-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy the file into the temp dir with its base name so the behavior matches directory mode.
	dest := filepath.Join(tmpDir, filepath.Base(inputFile))
	if err := copyFile(inputFile, dest); err != nil {
		return fmt.Errorf("stage file: %w", err)
	}

	// Reuse the existing implementation to ensure identical parsing/zip behavior.
	return input.ZipCorpusFromGoFuzzCases(tmpDir, outputName, verbose)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
