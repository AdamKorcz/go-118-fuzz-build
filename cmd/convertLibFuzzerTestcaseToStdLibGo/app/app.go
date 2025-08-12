package app

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
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
	"reflect"
	"strings"
	"testing"

	inputpkg "github.com/AdamKorcz/go-118-fuzz-build/input"
)

// Result holds one matched f.Fuzz(...) call found inside the requested function.
type Result struct {
	Line  int      // 1-based line number of the f.Fuzz call's '('
	Types []string // extracted types in call order, e.g. []{"[]byte","int","int","bool"}
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

		// Ensure selector receiver is an identifier matching one of the *testing.F params.
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

/*** JSON helpers and merge policy ***/

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
	err = os.Rename(tmp.Name(), path)
	if err != nil {
		return err
	}
	return os.Chmod(path, 0o644)
}

// MergeFuncTypesIntoJSON loads (or creates) jsonOut and adds funcName -> types
// if the key doesn't already exist. If the key exists and matches, it's a no-op.
// If the key exists with different types, it does not overwrite (no-op), by design.
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
			// No change
			return nil
		}
		// Different signature present: do not overwrite.
		return nil
	}

	m[funcName] = types
	return writeJSONMap(jsonOut, m)
}

// LoadTypesFromJSONFile returns the type list for funcName from a JSON map.
func LoadTypesFromJSONFile(path, funcName string) ([]string, error) {
	m, err := readJSONMap(path)
	if err != nil {
		return nil, err
	}
	types, ok := m[funcName]
	if !ok {
		return nil, fmt.Errorf("function %q not found in %s", funcName, path)
	}
	return types, nil
}

/*** Seed conversion using go-118-fuzz-build input.Source ***/

// ConvertSeedsToGoTests:
//  1) loads types for funcName from jsonPath,
//  2) builds an empty fuzz func: func(t *testing.T, ...types) {},
//  3) for each file in seedsDir, calls Source.CreateGoTestcase with the empty func,
//  4) writes the generated Go file to outDir, named as <md5(content)>.go.
//
// Returns number of files written.
func ConvertSeedsToGoTests(seedsDir, outDir, jsonPath, funcName string) (int, error) {
	types, err := LoadTypesFromJSONFile(jsonPath, funcName)
	if err != nil {
		return 0, err
	}
	emptyFn, err := MakeEmptyFuzzFunc(types)
	if err != nil {
		return 0, fmt.Errorf("build empty fuzz func: %w", err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(seedsDir)
	if err != nil {
		return 0, err
	}

	written := 0
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		p := filepath.Join(seedsDir, ent.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			return written, fmt.Errorf("read seed %s: %w", p, err)
		}

		// 1) Create Source from raw bytes.
		src := inputpkg.NewSource(data)

		// 2) Invoke CreateGoTestcase with our empty func.
		outVal := src.CreateGoTestcase(emptyFn.Interface(), reflect.ValueOf(new(testing.T)))

		// 3) Normalize to []byte.
		var outBytes []byte
		switch v := any(outVal).(type) {
		case string:
			outBytes = []byte(v)
		case []byte:
			outBytes = v
		default:
			return written, fmt.Errorf("unexpected CreateGoTestcase return type %T", v)
		}

		// 4) Name by md5 of content and write.
		sum := md5.Sum(outBytes)
		name := hex.EncodeToString(sum[:]) + ".go"
		dst := filepath.Join(outDir, name)

		if err := os.WriteFile(dst, outBytes, 0o644); err != nil {
			return written, fmt.Errorf("write %s: %w", dst, err)
		}
		written++
	}
	return written, nil
}

// MakeEmptyFuzzFunc constructs a reflect-made no-op function with signature:
//   func(t *testing.T, <types...>) {}
//
// It returns a reflect.Value holding the function, suitable to pass to
// input.Source.CreateGoTestcase.
func MakeEmptyFuzzFunc(paramTypes []string) (reflect.Value, error) {
	in := make([]reflect.Type, 0, len(paramTypes)+1)
	// First argument is always *testing.T
	in = append(in, reflect.TypeOf(new(testing.T)))
	for _, s := range paramTypes {
		rt, err := typeFromString(s)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("unsupported fuzz param type %q: %w", s, err)
		}
		in = append(in, rt)
	}
	fnType := reflect.FuncOf(in, nil, false)
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		// no-op body
		return nil
	})
	return fn, nil
}

// typeFromString maps a textual type (e.g., "[]byte", "int", "uint64")
// to a reflect.Type. It supports basic builtins and slices thereof.
func typeFromString(s string) (reflect.Type, error) {
	s = strings.TrimSpace(s)

	// Slices like []byte, []int, []rune, []string, etc.
	if strings.HasPrefix(s, "[]") {
		elem, err := typeFromString(s[2:])
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(elem), nil
	}

	switch s {
	case "string":
		return reflect.TypeOf(""), nil
	case "bool":
		return reflect.TypeOf(true), nil
	case "byte":
		return reflect.TypeOf(byte(0)), nil
	case "rune":
		return reflect.TypeOf(rune(0)), nil

	case "int":
		return reflect.TypeOf(int(0)), nil
	case "int8":
		return reflect.TypeOf(int8(0)), nil
	case "int16":
		return reflect.TypeOf(int16(0)), nil
	case "int32":
		return reflect.TypeOf(int32(0)), nil
	case "int64":
		return reflect.TypeOf(int64(0)), nil

	case "uint":
	 return reflect.TypeOf(uint(0)), nil
	case "uint8":
		return reflect.TypeOf(uint8(0)), nil
	case "uint16":
		return reflect.TypeOf(uint16(0)), nil
	case "uint32":
		return reflect.TypeOf(uint32(0)), nil
	case "uint64":
		return reflect.TypeOf(uint64(0)), nil
	case "uintptr":
		return reflect.TypeOf(uintptr(0)), nil

	case "float32":
		return reflect.TypeOf(float32(0)), nil
	case "float64":
		return reflect.TypeOf(float64(0)), nil
	}

	// Common fully spelled slice forms that may appear (kept for clarity).
	switch s {
	case "[]byte":
		return reflect.TypeOf([]byte(nil)), nil
	case "[]rune":
		return reflect.TypeOf([]rune(nil)), nil
	case "[]string":
		return reflect.TypeOf([]string(nil)), nil
	}

	return nil, fmt.Errorf("unrecognized type string %q", s)
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
			return base
		}
	}
	return base
}