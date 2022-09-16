package main

import (
	"go/ast"
)

type TFRewriter struct{}

func (v TFRewriter) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.SelectorExpr:
		switch n.Sel.Name {
		case "T", "F":
			// Patch
		default:
			return v
		}

		ident, ok := n.X.(*ast.Ident)
		if !ok {
			return v
		}

		if ident.Name != "testing" {
			return v
		}
		ident.Name = "go118fuzzbuildutils"
	}
	return v
}
