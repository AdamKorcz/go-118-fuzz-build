package main

import (
	"fmt"
	"go/token"
	"go/parser"

	"golang.org/x/tools/go/packages"
)

func rewriteTestingImports(pkgs []*packages.Package) error {
	for _, pkg := range pkgs {
		for _, file := range pkg.GoFile {
			err := rewriteTestingImport(file) {
				if err != nil {
					panic(err)
				}
			}
		}
	}

	// rewrite testing in imported packages
	packages.Visit(pkgs, rewriteTestingImport, nil)
	return nil
}

// Rewrites testing import of a single path
func rewriteTestingImport(path string) error {
	fmt.Println("Rewriting ", path)
	fsetCheck := token.NewFileSet()
	fCheck, err := parser.ParseFile(fsetCheck, GoFile, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}

	for _, imp := range fCheck.Imports {
		if imp.Path.Value == "testing" {
			imp.Path.Value = "github.com/AdamKorcz/go-118-fuzz-build/testing"
		}
	}
	return nil
}

// Rewrites testing import of a package
func rewriteImportTesting(pkg *packages.Package) bool {
	for _, file := range pkg.GoFile {
		err := rewriteTestingImport(file.)
	}
}