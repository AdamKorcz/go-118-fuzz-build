package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/AdamKorcz/go-118-fuzz-build/cmd/convertLibFuzzerTestcaseToStdLibGo/app"
)

func main() {
	// Modes: exactly one must be true.
	getParams := flag.Bool("get-params", false, "print the fuzz parameter types (JSON array) to stdout")
	writeParams := flag.Bool("write-params", false, "merge the fuzz parameter types into the JSON file (requires -json-out)")
	convertSeeds := flag.Bool("convert-seeds", false, "convert raw seed files into stdlib Go fuzz testcases (requires -params-json, -seeds-dir, -out-dir)")

	// Common inputs
	fileFlag := flag.String("file", "", "path to a single .go file to scan (required for -get-params and -write-params)")
	funcFlag := flag.String("fuzzer-func", "", "name of the function that calls f.Fuzz(...) (required)")

	// Params JSON for write/convert
	jsonOut := flag.String("json-out", "", "path to output JSON file (created or merged) (required for -write-params)")
	paramsJSON := flag.String("params-json", "", "path to JSON file mapping function name to fuzz param types (required for -convert-seeds). If omitted and -json-out is set, that path will be used.")

	// Behavior
	firstOnly := flag.Bool("first", true, "use only the first matching f.Fuzz in the function")

	// Seed conversion I/O
	seedsDir := flag.String("seeds-dir", "", "directory containing raw seed files (required for -convert-seeds)")
	outDir := flag.String("out-dir", "", "directory to write generated *_test.go files (required for -convert-seeds)")

	flag.Parse()

	// Validate exactly one mode selected
	modeCount := 0
	for _, b := range []bool{*getParams, *writeParams, *convertSeeds} {
		if b {
			modeCount++
		}
	}
	if modeCount != 1 {
		fatalf("error: specify exactly one of -get-params, -write-params, or -convert-seeds")
	}

	// Validate required function name
	if *funcFlag == "" {
		fatalf("error: -fuzzer-func is required")
	}

	switch {
	case *getParams:
		if *fileFlag == "" {
			fatalf("error: -file is required for -get-params")
		}
		results, err := app.AnalyzeFile(*fileFlag, *funcFlag)
		if err != nil {
			fatalf("%v", err)
		}
		types := app.ChooseTypes(results, *firstOnly)
		if len(types) == 0 {
			fatalf("no parameter types found")
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(types); err != nil {
			fatalf("encode error: %v", err)
		}

	case *writeParams:
		if *fileFlag == "" || *jsonOut == "" {
			fatalf("error: -file and -json-out are required for -write-params")
		}
		results, err := app.AnalyzeFile(*fileFlag, *funcFlag)
		if err != nil {
			fatalf("%v", err)
		}
		types := app.ChooseTypes(results, *firstOnly)
		if len(types) == 0 {
			fatalf("no parameter types found")
		}
		if err := app.MergeFuncTypesIntoJSON(*jsonOut, *funcFlag, types); err != nil {
			fatalf("merge error: %v", err)
		}
		fmt.Printf("Saved %q -> %v into %s\n", *funcFlag, types, *jsonOut)

	case *convertSeeds:
		// Allow -params-json to fall back to -json-out if not provided explicitly.
		if *paramsJSON == "" && *jsonOut != "" {
			*paramsJSON = *jsonOut
		}
		if *paramsJSON == "" || *seedsDir == "" || *outDir == "" {
			fatalf("error: -params-json, -seeds-dir, and -out-dir are required for -convert-seeds")
		}
		n, err := app.ConvertSeedsToGoTests(*seedsDir, *outDir, *paramsJSON, *funcFlag)
		if err != nil {
			fatalf("convert-seeds error: %v", err)
		}
		fmt.Printf("Generated %d fuzz testcases in %s\n", n, *outDir)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}