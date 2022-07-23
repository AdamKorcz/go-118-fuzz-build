package main

import (
	"go/ast"
	"go/token"
)

type LogRewriter struct {
	fset *token.FileSet
	file *ast.File
}

func (walker *LogRewriter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CallExpr:
		if aa, ok := n.Fun.(*ast.SelectorExpr); ok {
			if _, ok := aa.X.(*ast.Ident); ok {
				if aa.X.(*ast.Ident).Name == "t" {
					if isTestFatal(aa.Sel.Name) {
						aa.X.(*ast.Ident).Name = "testingtypes"
					}
				}
			}
		}
	}
	return walker
}

func rewriteLogStatements(path string, astFile *ast.File, fset *token.FileSet) {
	walker := &LogRewriter{file: astFile, fset: fset}

	ast.Walk(walker, walker.file)
}

func isTestFatal(name string) bool {
	switch name {
	case "Error":
		return true
	case "Errorf":
		return true
	case "Fatal":
		return true
	case "Fatalf":
		return true
	case "Log":
		return true
	case "Logf":
		return true
	case "Setenv":
		return true
	}
	return false
}
