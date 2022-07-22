//go:build go1.18

package main

import (
	"bytes"
	"testing"
	"go/parser"
	"go/printer"
	"fmt"

	"go/token"
)

// Test that RangeNodes returns a slice of nodes with contiguous coverage.
// https://github.com/transparency-dev/merkle/blob/main/docs/compact_ranges.md#definition
func TestMain(T *testing.T) {
	fuzzerPath := "../testdata/first_fuzz_test.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fuzzerPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	addImport(f, fset)

	buf := new(bytes.Buffer)
	err = printer.Fprint(buf, fset, f)
	if err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
	return
}