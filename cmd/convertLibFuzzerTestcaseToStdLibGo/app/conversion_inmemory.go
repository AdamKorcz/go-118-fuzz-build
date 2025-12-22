package app

import (
	"fmt"
	"reflect"
	"testing"

	inputpkg "github.com/AdamKorcz/go-118-fuzz-build/input"
)

// ConvertLibFuzzerToGoInMemory converts libFuzzer corpus data to Go corpus format in memory
// without any file I/O operations.
func ConvertLibFuzzerToGoInMemory(rawData []byte, typesList []string) ([]byte, error) {
	// Build empty fuzz function with the given signature
	emptyFn, err := MakeEmptyFuzzFunc(typesList)
	if err != nil {
		return nil, fmt.Errorf("build empty fuzz func: %w", err)
	}

	// Create Source from raw bytes
	src := inputpkg.NewSource(rawData)

	// Invoke CreateGoTestcaseWithBoundaries for multi-parameter fuzzer support
	outVal := src.CreateGoTestcaseWithBoundaries(emptyFn.Interface(), reflect.ValueOf(new(testing.T)))

	// Normalize to []byte
	var outBytes []byte
	switch v := any(outVal).(type) {
	case string:
		outBytes = []byte(v)
	case []byte:
		outBytes = v
	default:
		return nil, fmt.Errorf("unexpected CreateGoTestcase return type %T", v)
	}

	return outBytes, nil
}

// ConvertGoToLibFuzzerInMemory converts Go corpus format data back to libFuzzer format in memory
// without any file I/O operations.
func ConvertGoToLibFuzzerInMemory(goCorpusData []byte) ([]byte, error) {
	return inputpkg.ParseGoTestcase(string(goCorpusData))
}

// RoundtripConversion performs a complete roundtrip conversion in memory and validates consistency.
// Returns the three formats: original libFuzzer, Go corpus, and roundtripped libFuzzer.
func RoundtripConversion(rawData []byte, typesList []string) (original, goCorpus, roundtripped []byte, err error) {
	// Step 1: Convert libFuzzer → Go
	goCorpus, err = ConvertLibFuzzerToGoInMemory(rawData, typesList)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("libFuzzer→Go conversion failed: %w", err)
	}

	// Step 2: Convert Go → libFuzzer
	roundtripped, err = ConvertGoToLibFuzzerInMemory(goCorpus)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Go→libFuzzer conversion failed: %w", err)
	}

	return rawData, goCorpus, roundtripped, nil
}
