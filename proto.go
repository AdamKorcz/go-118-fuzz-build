package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

const protoPkgAlias = "gofuzzprotopb"

const (
	protoFormatBinary = "binary"
	protoFormatText   = "text"
)

type ProtoTarget struct {
	ImportPath string
	TypeName   string
	Format     string
}

func findProtoTarget(pkgs []*packages.Package, funcName, format string) (*ProtoTarget, error) {
	switch format {
	case protoFormatBinary, protoFormatText:
	default:
		return nil, fmt.Errorf("-proto_format must be %q or %q, got %q", protoFormatBinary, protoFormatText, format)
	}
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fuzzFunc, ok := decl.(*ast.FuncDecl)
				if !ok || fuzzFunc.Name.Name != funcName || fuzzFunc.Body == nil {
					continue
				}
				return protoTargetFromFuzzFunc(pkg, fuzzFunc, format)
			}
		}
	}
	return nil, fmt.Errorf("could not find the fuzz func %q in the loaded packages", funcName)
}

func protoTargetFromFuzzFunc(pkg *packages.Package, fuzzFunc *ast.FuncDecl, format string) (*ProtoTarget, error) {
	funcName := fuzzFunc.Name.Name
	callback := findFuzzCallback(fuzzFunc)
	if callback == nil {
		return nil, fmt.Errorf("found no 'f.Fuzz(func(t *testing.T, msg *pb.MyMessage){...})' call in the body of %s", funcName)
	}
	params := paramTypeExprs(callback.Type.Params)
	if len(params) != 2 {
		return nil, fmt.Errorf("the f.Fuzz callback of %s takes %d parameters, but a proto fuzzer must take "+
			"exactly two: func(t *testing.T, msg *pb.MyMessage)", funcName, len(params))
	}
	ptr, ok := pkg.TypesInfo.TypeOf(params[1]).(*types.Pointer)
	if !ok {
		return nil, fmt.Errorf("the second parameter of the f.Fuzz callback of %s is not a pointer to a "+
			"protoc-generated message", funcName)
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok || named.Obj().Pkg() == nil || !hasProtoReflect(ptr) {
		return nil, fmt.Errorf("%s is not a protoc-generated message: it has no ProtoReflect() method", ptr.String())
	}
	return &ProtoTarget{
		ImportPath: named.Obj().Pkg().Path(),
		TypeName:   named.Obj().Name(),
		Format:     format,
	}, nil
}

func findFuzzCallback(fuzzFunc *ast.FuncDecl) *ast.FuncLit {
	var callback *ast.FuncLit
	ast.Inspect(fuzzFunc.Body, func(n ast.Node) bool {
		if callback != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Fuzz" || len(call.Args) != 1 {
			return true
		}
		callback, _ = call.Args[0].(*ast.FuncLit)
		return callback == nil
	})
	return callback
}

func paramTypeExprs(params *ast.FieldList) []ast.Expr {
	exprs := make([]ast.Expr, 0)
	if params == nil {
		return exprs
	}
	for _, field := range params.List {
		names := len(field.Names)
		if names == 0 {
			names = 1
		}
		for i := 0; i < names; i++ {
			exprs = append(exprs, field.Type)
		}
	}
	return exprs
}

func hasProtoReflect(t types.Type) bool {
	return types.NewMethodSet(t).Lookup(nil, "ProtoReflect") != nil
}

func (p *ProtoTarget) fillData(d *Data, localPkgPath, localAlias string) {
	d.Proto = true
	d.ProtoFormat = p.Format

	switch {
	case p.ImportPath != localPkgPath:
		d.ProtoImport = "\t" + protoPkgAlias + " " + strconv.Quote(p.ImportPath) + "\n"
		d.ProtoType = protoPkgAlias + "." + p.TypeName
	case localAlias == "":
		d.ProtoType = p.TypeName
	default:
		d.ProtoType = localAlias + "." + p.TypeName
	}

	codecs := []string{"google.golang.org/protobuf/proto"}
	d.ProtoUnmarshal, d.ProtoMarshal = "proto.Unmarshal", "proto.Marshal"
	if p.Format == protoFormatText {
		codecs = append(codecs, "google.golang.org/protobuf/encoding/prototext")
		d.ProtoUnmarshal, d.ProtoMarshal = "prototext.Unmarshal", "prototext.Marshal"
	}
	var imports strings.Builder
	for _, codec := range codecs {
		imports.WriteString("\t" + strconv.Quote(codec) + "\n")
	}
	d.ProtoCodecImports = imports.String()
}
