package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AdamKorcz/go-118-fuzz-build/cmd/libFuzzerCorpusToStdLibCorpus/app"
)

func main() {
	inputDir := flag.String("in", "", "Path to input directory containing test case files")
	outputDir := flag.String("out", "", "Path to output directory for converted test cases")
	flag.Parse()

	if *inputDir == "" || *outputDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -in <input_directory> -out <output_directory>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := app.ConvertDirectory(*inputDir, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}