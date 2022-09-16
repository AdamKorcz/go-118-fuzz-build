package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	fuzzerPath = flag.String("path", "", "path to fuzzer")
)

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func addImport(astFile *ast.File, fset *token.FileSet, addTestingtypes bool) {
	if addTestingtypes {
		astutil.AddImport(fset, astFile, "github.com/AdamKorcz/go-118-fuzz-build/testingtypes")
	}
	astutil.AddNamedImport(fset, astFile, "go118fuzzbuildutils", "github.com/AdamKorcz/go-118-fuzz-build/utils")
}

func getStringVersion(start, end token.Pos, src []byte) string {
	return string(src[start-1 : end-1])
}

func main() {
	flag.Parse()
	if !isFlagSet("path") {
		fmt.Println("Please provide a path to the fuzzer")
		os.Exit(1)
	}
	_, err := os.Stat(*fuzzerPath)
	if err != nil {
		fmt.Printf("ERROR: %s does not exist\n", *fuzzerPath)
		os.Exit(1)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *fuzzerPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	ast.Walk(TFRewriter{}, f)

	addTestingtypes := rewriteLogStatements(*fuzzerPath, f, fset)

	addImport(f, fset, addTestingtypes)

	buf := new(bytes.Buffer)
	err = printer.Fprint(buf, fset, f)
	if err != nil {
		panic(err)
	}
	//fmt.Println(buf.String())

	err = os.Remove(*fuzzerPath)
	if err != nil {
		panic(err)
	}

	fo, err := os.Create(*fuzzerPath)
	if err != nil {
		panic(err)
	}
	defer fo.Close()

	_, err = fo.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}

}
