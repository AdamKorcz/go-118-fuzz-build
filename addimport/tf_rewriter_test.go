package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"testing"
)

func TestRewriteTFReferences(t *testing.T) {
	fuzzerPath := "../testdata/first_fuzz_test.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fuzzerPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	ast.Walk(TFRewriter{}, f)
	buf := new(bytes.Buffer)
	err = printer.Fprint(buf, fset, f)
	if err != nil {
		panic(err)
	}
	t.Log(buf.String())
}
