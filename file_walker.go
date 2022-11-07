package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/parser"
	"go/printer"
	"os"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ast/astutil"
)

func rewriteTestingImports(pkgs []*packages.Package, fuzzName string) error {
	//var fuzzFilepath string

	// First find file with fuzz harness
	for _, pkg := range pkgs {
		for _, file := range pkg.GoFiles {
			//fmt.Println(file)
			err := rewriteTestingImport(file)
			if err != nil {
				panic(err)
			}
		}
	}

	// rewrite testing in imported packages
	packages.Visit(pkgs, rewriteImportTesting, nil)

	for _, pkg := range pkgs {
		for _, file := range pkg.GoFiles {
			err := rewriteFuzzer(file, fuzzName)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func rewriteFuzzer(path, fuzzerName string) error {
	var fileHasOurHarness bool // to determine whether we should rewrite filename
	fileHasOurHarness = false

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}
	for _, decl := range f.Decls {
		if _, ok := decl.(*ast.FuncDecl); ok {
			if decl.(*ast.FuncDecl).Name.Name == fuzzerName {
				fileHasOurHarness = true
			}
		}
	}

	if fileHasOurHarness {
		// Replace import path
		astutil.DeleteImport(fset, f, "testing")
		astutil.AddImport(fset, f, "github.com/AdamKorcz/go-118-fuzz-build/testing")
	}

	// Rewrite filename
	if fileHasOurHarness {
		fmt.Println("WWWWWWWWWWWWWWWWWWE HAVE OUR FUZZER")

		var buf bytes.Buffer
		printer.Fprint(&buf, fset, f)
		
		os.Remove(path)
		newFile, err := os.Create(path+"_fuzz.go")
		if err != nil {
			panic(err)
		}
		defer newFile.Close()
		newFile.Write(buf.Bytes())
		b, err := os.ReadFile(path+"_fuzz.go")
		if err != nil {
			panic(err)
		}		
		fmt.Println(string(b))
	}
	return nil

}

// Rewrites testing import of a single path
func rewriteTestingImport(path string) error {
	//fmt.Println("Rewriting ", path)
	fsetCheck := token.NewFileSet()
	fCheck, err := parser.ParseFile(fsetCheck, path, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}

	// First check if the import already exists
	// Return if it does.
	for _, imp := range fCheck.Imports {
		if imp.Path.Value == "github.com/AdamKorcz/go-118-fuzz-build/testing" {
			return nil
		}
	}

	// Replace import path
	for _, imp := range fCheck.Imports {
		if imp.Path.Value == "testing" {
			imp.Path.Value = "github.com/AdamKorcz/go-118-fuzz-build/testing"
		}
	}
	return nil
}

// Rewrites testing import of a package
func rewriteImportTesting(pkg *packages.Package) bool {
	for _, file := range pkg.GoFiles {
		err := rewriteTestingImport(file)
		if err != nil {
			panic(err)
		}
	}
	return true
}

// Checks whether a fuzz test exists in a given file
func rewriteFuzzerImports(path, fuzzName string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}
	for _, decl := range f.Decls {
		if _, ok := decl.(*ast.FuncDecl); ok {
			if decl.(*ast.FuncDecl).Name.Name == fuzzName {
				// First rewrite testing.F

			}
		}
	}
	return nil
}