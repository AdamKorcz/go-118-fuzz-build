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

func addImport(astFile *ast.File) {
	path := "github.com/AdamKorcz/go-118-fuzz-build/utils"
	name := "go118fuzzbuildutils"
	newImport := &ast.ImportSpec{
		Name: ast.NewIdent(name),
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf("%q", path),
		},
	}
	impDecl := &ast.GenDecl{
		Lparen: astFile.Name.End(),
		Tok:    token.IMPORT,
		Specs: []ast.Spec{
			newImport,
		},
		Rparen: astFile.Name.End(),
	}
	_, _ = newImport, impDecl
	astFile.Decls = append(astFile.Decls, nil)
	copy(astFile.Decls[1:], astFile.Decls[0:])
	astFile.Decls[0] = impDecl
	astFile.Imports = append(astFile.Imports, newImport)
}

func getStringVersion(start, end token.Pos, src  []byte) string {
    return string(src[start-1:end-1])
}

func main() {
	flag.Parse()
	if !isFlagSet("path") {
		fmt.Println("Please provide a path to the fuzzer")
		os.Exit(1)
	}
	_, err := os.Stat(*fuzzerPath)
	if err != nil {
		fmt.Printf("ERROR: %s does not exist\n", fuzzerPath)
		os.Exit(1)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *fuzzerPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	
	addImport(f)

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