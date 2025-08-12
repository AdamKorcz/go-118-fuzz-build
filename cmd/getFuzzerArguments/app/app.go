package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
)

// Result holds one matched f.Fuzz(...) call found inside the requested function.
type Result struct {
	Line  int      // 1-based line number of the f.Fuzz call's '('
	Types []string // extracted types, e.g. []{"[]byte","int","int","bool"}
}

// AnalyzeFile parses a single .go file and extracts fuzz arg types from
// f.Fuzz(...) calls inside function funcName, ensuring the receiver is a *testing.F.
func AnalyzeFile(path, funcName string) ([]Result, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}
	return analyzeAST(fset, file, funcName)
}

// AnalyzeSource is handy for unit tests (or callers with in-memory source).
func AnalyzeSource(src []byte, funcName string) ([]Result, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "inmem.go", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}
	return analyzeAST(fset, file, funcName)
}

// ChooseTypes selects which types list to persist when multiple f.Fuzz calls exist.
// If firstOnly is true, it returns the first. Otherwise it prefers a unanimous list;
// if they differ, it still returns the first and leaves it to the caller to warn/log.
func ChooseTypes(results []Result, firstOnly bool) []string {
	if len(results) == 0 {
		return nil
	}
	if firstOnly {
		return results[0].Types
	}
	base := results[0].Types
	for i := 1; i < len(results); i++ {
		if !equalStrings(base, results[i].Types) {
			// Caller may choose to warn; we just return the first.
			return base
		}
	}
	return base
}

func analyzeAST(fset *token.FileSet, file *ast.File, funcName string) ([]Result, error) {
	fn := findFuncDecl(file, funcName)
	if fn == nil || fn.Body == nil {
		return nil, fmt.Errorf("no function %q with body found", funcName)
	}

	// Collect names of parameters whose type is *testing.F for this function.
	fuzzParamNames := namesOfTestingFParams(fn)
	if len(fuzzParamNames) == 0 {
		return nil, fmt.Errorf("function %q has no parameter of type *testing.F", funcName)
	}

	var results []Result
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Fuzz" {
			return true
		}

		// Ensure selector receiver is an identifier matching one of the *testing.F params
		id, ok := sel.X.(*ast.Ident)
		if !ok || !fuzzParamNames[id.Name] {
			return true
		}

		// Find the func literal argument: f.Fuzz(func(t *testing.T, ...){...})
		for _, arg := range call.Args {
			fnLit, ok := arg.(*ast.FuncLit)
			if !ok || fnLit.Type == nil || fnLit.Type.Params == nil {
				continue
			}
			types := extractParamTypes(fnLit.Type.Params.List)
			if len(types) == 0 {
				continue
			}
			pos := fset.Position(call.Lparen)
			results = append(results, Result{
				Line:  pos.Line,
				Types: types,
			})
		}
		return true
	})

	if len(results) == 0 {
		return nil, fmt.Errorf("no matching f.Fuzz(func(...){...}) found inside %q", funcName)
	}
	return results, nil
}

func findFuncDecl(f *ast.File, name string) *ast.FuncDecl {
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name != nil && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

func namesOfTestingFParams(fn *ast.FuncDecl) map[string]bool {
	out := make(map[string]bool)
	if fn.Type == nil || fn.Type.Params == nil {
		return out
	}
	for _, field := range fn.Type.Params.List {
		if isTestingF(field.Type) {
			for _, name := range field.Names {
				out[name.Name] = true
			}
		}
	}
	return out
}

// extractParamTypes returns fuzz func parameter types after skipping the
// initial t *testing.T (if present) and expanding grouped params (a, b int -> int, int).
func extractParamTypes(fields []*ast.Field) []string {
	start := 0
	if len(fields) > 0 && isTestingT(fields[0].Type) {
		start = 1
	}
	var out []string
	for i := start; i < len(fields); i++ {
		typ := exprString(fields[i].Type)
		count := 1
		if len(fields[i].Names) > 0 {
			count = len(fields[i].Names)
		}
		for j := 0; j < count; j++ {
			out = append(out, typ)
		}
	}
	return out
}

func isTestingT(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.StarExpr:
		return isTestingT(x.X)
	case *ast.SelectorExpr:
		if x.Sel != nil && x.Sel.Name == "T" {
			if pkg, ok := x.X.(*ast.Ident); ok && pkg.Name == "testing" {
				return true
			}
		}
	}
	return false
}

func isTestingF(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.StarExpr:
		return isTestingF(x.X)
	case *ast.SelectorExpr:
		if x.Sel != nil && x.Sel.Name == "F" {
			if pkg, ok := x.X.(*ast.Ident); ok && pkg.Name == "testing" {
				return true
			}
		}
	}
	return false
}

func exprString(e ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

/*** JSON merge helpers ***/

func readJSONMap(path string) (map[string][]string, error) {
	m := make(map[string][]string)

	// If file does not exist, return empty map.
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return m, nil
		}
		return nil, err
	}

	// Read and unmarshal.
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return m, nil
}

func writeJSONMap(path string, m map[string][]string) error {
	// Ensure directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Write to a temp file then rename.
	tmp, err := os.CreateTemp(dir, ".tmp-json-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// MergeFuncTypesIntoJSON loads (or creates) jsonOut and adds funcName -> types
// if the key doesn't already exist. If the key exists and matches, it's a no-op.
// If the key exists with different types, it does not overwrite (returns nil with a message-worthy condition).
func MergeFuncTypesIntoJSON(jsonOut, funcName string, types []string) error {
	if len(types) == 0 {
		return fmt.Errorf("no types to merge for %q", funcName)
	}
	m, err := readJSONMap(jsonOut)
	if err != nil {
		return err
	}

	if existing, ok := m[funcName]; ok {
		if equalStrings(existing, types) {
			// no change
			return nil
		}
		// per requirement: do not overwrite if different — treat as no-op
		return nil
	}

	m[funcName] = types
	return writeJSONMap(jsonOut, m)
}