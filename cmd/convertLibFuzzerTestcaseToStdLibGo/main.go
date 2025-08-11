package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AdamKorcz/go-118-fuzz-build/cmd/convertLibFuzzerTestcaseToStdLibGo/app"
)

func main() {
	fileFlag := flag.String("file", "", "path to a single .go file to scan (required)")
	funcFlag := flag.String("fuzzer-func", "", "name of the function that calls f.Fuzz(...) (required)")
	jsonOut := flag.String("json-out", "", "path to output JSON file (created or merged) (required)")
	firstOnly := flag.Bool("first", true, "use only the first matching f.Fuzz in the function")
	flag.Parse()

	if *fileFlag == "" || *funcFlag == "" || *jsonOut == "" {
		fmt.Fprintln(os.Stderr, "error: -file, -fuzzer-func, and -json-out are required")
		flag.Usage()
		os.Exit(2)
	}

	results, err := app.AnalyzeFile(*fileFlag, *funcFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	types := app.ChooseTypes(results, *firstOnly)

	if err := app.MergeFuncTypesIntoJSON(*jsonOut, *funcFlag, types); err != nil {
		fmt.Fprintln(os.Stderr, "merge error:", err)
		os.Exit(1)
	}

	fmt.Printf("Saved %q -> %v into %s\n", *funcFlag, types, *jsonOut)
}