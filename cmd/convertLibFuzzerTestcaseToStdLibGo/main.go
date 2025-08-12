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
	convertSeeds := flag.Bool("convert-seeds", false, "convert raw seed files into stdlib Go fuzz corpus files (requires -params-json, -seeds-dir, -out-dir)")

	// Common inputs
	fileFlag := flag.String("file", "", "path to a single .go file to scan (required for -get-params and -write-params)")
	funcFlag := flag.String("fuzzer-func", "", "name of the function that calls f.Fuzz(...) (required for -get-params and -write-params; used as default JSON key)")

	// JSON key override
	fuzzerBinaryName := flag.String("fuzzerBinaryName", "", "override key used to read/write params in the JSON (defaults to -fuzzer-func)")

	// Params JSON for write/convert
	jsonOut := flag.String("json-out", "", "path to output JSON file (created or merged) (required for -write-params)")
	paramsJSON := flag.String("params-json", "", "path to JSON file mapping name -> fuzz param types (required for -convert-seeds). If omitted and -json-out is set, that path will be used.")

	// Behavior
	firstOnly := flag.Bool("first", true, "use only the first matching f.Fuzz in the function")

	// Seed conversion I/O
	seedsDir := flag.String("seeds-dir", "", "directory containing raw seed files (required for -convert-seeds)")
	outDir := flag.String("out-dir", "", "directory to write generated corpus files (required for -convert-seeds)")

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

	// Determine the JSON key name (used for write/read with the params JSON).
	// If -fuzzerBinaryName is set, it overrides; otherwise use -fuzzer-func.
	jsonKey := *fuzzerBinaryName
	if jsonKey == "" {
		jsonKey = *funcFlag
	}

	switch {
	case *getParams:
		// For get-params we only analyze code; -fuzzer-func and -file are required.
		if *fileFlag == "" || *funcFlag == "" {
			fatalf("error: -file and -fuzzer-func are required for -get-params")
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
		// For write-params we both analyze code and write to JSON.
		if *fileFlag == "" || *funcFlag == "" || *jsonOut == "" {
			fatalf("error: -file, -fuzzer-func, and -json-out are required for -write-params")
		}
		if jsonKey == "" {
			// Should not happen given the above, but be explicit.
			fatalf("error: JSON key is empty; set -fuzzer-func or -fuzzerBinaryName")
		}
		results, err := app.AnalyzeFile(*fileFlag, *funcFlag)
		if err != nil {
			fatalf("%v", err)
		}
		types := app.ChooseTypes(results, *firstOnly)
		if len(types) == 0 {
			fatalf("no parameter types found")
		}
		if err := app.MergeFuncTypesIntoJSON(*jsonOut, jsonKey, types); err != nil {
			fatalf("merge error: %v", err)
		}
		fmt.Printf("Saved %q -> %v into %s\n", jsonKey, types, *jsonOut)

	case *convertSeeds:
		// For convert-seeds we only read from JSON and generate corpus files.
		// Allow -params-json to fall back to -json-out if not provided explicitly.
		if *paramsJSON == "" && *jsonOut != "" {
			*paramsJSON = *jsonOut
		}
		if *paramsJSON == "" || *seedsDir == "" || *outDir == "" {
			fatalf("error: -params-json, -seeds-dir, and -out-dir are required for -convert-seeds")
		}
		// For convert-seeds, the JSON key can be provided via -fuzzerBinaryName;
		// if not set, we fall back to -fuzzer-func. Require one of them.
		if jsonKey == "" {
			fatalf("error: specify either -fuzzerBinaryName or -fuzzer-func for -convert-seeds (used as the JSON key)")
		}
		n, err := app.ConvertSeedsToGoTests(*seedsDir, *outDir, *paramsJSON, jsonKey)
		if err != nil {
			fatalf("convert-seeds error: %v", err)
		}
		fmt.Printf("Generated %d fuzz corpus files in %s\n", n, *outDir)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}