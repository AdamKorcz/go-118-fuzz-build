//go:build go1.18

package main

import (
	"bytes"
	//"fmt"

	"go/parser"

	"go/printer"
	"go/token"
	"testing"
)

func TestRewriteLogStatements(t *testing.T) {
	fuzzerPath := "../testdata/first_fuzz_test.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fuzzerPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	rewriteLogStatements(fuzzerPath, f, fset)
	buf := new(bytes.Buffer)
	err = printer.Fprint(buf, fset, f)
	if err != nil {
		panic(err)
	}
	t.Log("HEre")
	t.Log(buf.String())
}
