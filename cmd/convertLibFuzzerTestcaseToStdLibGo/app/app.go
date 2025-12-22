package app

import (
	"bytes"
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	inputpkg "github.com/AdamKorcz/go-118-fuzz-build/input"
	sitter "github.com/smacker/go-tree-sitter"
	tsgo "github.com/smacker/go-tree-sitter/golang"
	"golang.org/x/tools/go/packages"
)

// Caching of loaded/type-checked worlds.
var (
	cacheMu          sync.RWMutex
	moduleWorldCache = make(map[string]*astWorld) // key: abs module root (or "dir:"+absDir when no go.mod)
	sourceWorldCache = make(map[string]*astWorld) // key: "src:"+md5(src)
)

/***************
 * Analysis API
 ***************/

// Result holds one matched f.Fuzz(...) call found inside the requested function.
type Result struct {
	Line  int      // 1-based line number of the f.Fuzz call's '('
	Types []string // extracted types in call order, e.g. []{"[]byte","int","int","bool"}
}

// AnalyzeFile analyzes the function/method named funcName that is defined in the
// file at 'path'. It loads and type-checks the entire module (./...) so traversal
// can follow calls across packages within the module.
//
// If no module root (go.mod) is found, it falls back to loading just the package
// containing 'path' and its deps (without syntax for deps), which limits traversal.
func AnalyzeFile(path string, funcName string) ([]Result, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}
	dir := filepath.Dir(abs)
	modRoot, hasMod := findModuleRoot(dir)

	// Cache key
	cacheKey := ""
	if hasMod {
		cacheKey = modRoot
	} else {
		cacheKey = "dir:" + dir
	}

	// Try cache
	cacheMu.RLock()
	world := moduleWorldCache[cacheKey]
	cacheMu.RUnlock()

	// Build world if needed
	if world == nil {
		mode := packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedModule |
			packages.NeedImports |
			packages.NeedDeps

		cfg := &packages.Config{Mode: mode, Tests: true}
		var patterns []string
		if hasMod {
			cfg.Dir = modRoot
			patterns = []string{"./..."}
		} else {
			cfg.Dir = dir
			patterns = []string{"file=" + abs}
		}

		pkgs, err := packages.Load(cfg, patterns...)
		if err != nil {
			return nil, fmt.Errorf("packages.Load: %w", err)
		}
		if packages.PrintErrors(pkgs) > 0 {
			// proceed best-effort; target file may still be fine
		}

		world, err = buildWorldFromPackages(pkgs)
		if err != nil {
			return nil, err
		}

		cacheMu.Lock()
		moduleWorldCache[cacheKey] = world
		cacheMu.Unlock()
	}

	return analyzeAST(world, abs, funcName)
}

// AnalyzeSource is handy for unit tests (or callers with in-memory source).
func AnalyzeSource(src []byte, funcName string) ([]Result, error) {
	const pseudo = "<src>"

	// Cache by content hash
	sum := md5.Sum(src)
	key := "src:" + hex.EncodeToString(sum[:])

	cacheMu.RLock()
	world := sourceWorldCache[key]
	cacheMu.RUnlock()

	if world == nil {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, pseudo, src, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse: %w", err)
		}
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}
		conf := &types.Config{
			Importer: importer.Default(),
		}
		pkg, err := conf.Check(file.Name.Name, fset, []*ast.File{file}, info)
		if err != nil {
			return nil, fmt.Errorf("type check: %w", err)
		}

		world = &astWorld{
			filesByPath: map[string]*ast.File{pseudo: file},
			srcByFile:   map[string][]byte{pseudo: src},
			tfByFile:    map[string]*token.File{pseudo: fset.File(file.Pos())},
			fsetByFile:  map[string]*token.FileSet{pseudo: fset},
			infoByFile:  map[string]*types.Info{pseudo: info},
			pkgByFile:   map[string]*types.Package{pseudo: pkg},
		}

		cacheMu.Lock()
		sourceWorldCache[key] = world
		cacheMu.Unlock()
	}

	return analyzeAST(world, pseudo, funcName)
}

/*************************
 * Internal analysis core
 *************************/

type astWorld struct {
	filesByPath map[string]*ast.File      // path or pseudo -> *ast.File
	srcByFile   map[string][]byte         // path -> source bytes
	tfByFile    map[string]*token.File    // path -> token.File for offset/pos mapping
	fsetByFile  map[string]*token.FileSet // path -> file's FileSet (for Position)
	infoByFile  map[string]*types.Info    // path -> package's TypesInfo for that file
	pkgByFile   map[string]*types.Package // path -> *types.Package
}

type funcNode struct {
	obj   *types.Func
	decl  *ast.FuncDecl
	fname string // path or pseudo
}

// buildWorldFromPackages builds an astWorld from packages.Load results.
func buildWorldFromPackages(pkgs []*packages.Package) (*astWorld, error) {
	w := &astWorld{
		filesByPath: make(map[string]*ast.File),
		srcByFile:   make(map[string][]byte),
		tfByFile:    make(map[string]*token.File),
		fsetByFile:  make(map[string]*token.FileSet),
		infoByFile:  make(map[string]*types.Info),
		pkgByFile:   make(map[string]*types.Package),
	}

	for _, p := range pkgs {
		if p == nil || p.Fset == nil || p.TypesInfo == nil {
			continue
		}
		for _, file := range p.Syntax {
			if file == nil {
				continue
			}
			filename := p.Fset.Position(file.Pos()).Filename
			abs, _ := filepath.Abs(filename)

			w.filesByPath[abs] = file
			w.tfByFile[abs] = p.Fset.File(file.Pos())
			w.fsetByFile[abs] = p.Fset
			w.infoByFile[abs] = p.TypesInfo
			w.pkgByFile[abs] = p.Types

			// Source bytes
			if _, ok := w.srcByFile[abs]; !ok {
				data, err := os.ReadFile(abs)
				if err == nil {
					w.srcByFile[abs] = data
				} else {
					// best effort; not fatal
					w.srcByFile[abs] = nil
				}
			}
		}
	}
	if len(w.filesByPath) == 0 {
		return nil, fmt.Errorf("no files with syntax loaded")
	}
	return w, nil
}

func analyzeAST(world *astWorld, filename string, funcName string) ([]Result, error) {
	file := world.filesByPath[filename]
	if file == nil {
		return nil, fmt.Errorf("file %s not found in loaded packages", filename)
	}

	// Resolve the root function/method by name INSIDE the given file.
	rootDecls := findFuncDeclsByName(file, funcName)
	if len(rootDecls) == 0 {
		return nil, fmt.Errorf("function or method %q not found in %s", funcName, filename)
	}
	if len(rootDecls) > 1 {
		return nil, fmt.Errorf("ambiguous: multiple decls named %q in %s", funcName, filename)
	}
	rootDecl := rootDecls[0]
	if rootDecl.Body == nil {
		return nil, fmt.Errorf("function %q has no body", funcName)
	}
	info := world.infoByFile[filename]
	if info == nil {
		return nil, fmt.Errorf("types.Info not found for %s", filename)
	}
	rootObj, _ := info.Defs[rootDecl.Name].(*types.Func)
	if rootObj == nil {
		return nil, fmt.Errorf("failed to resolve *types.Func for %q", funcName)
	}
	rootNode := &funcNode{obj: rootObj, decl: rootDecl, fname: filename}

	// Tree-sitter setup
	lang := tsgo.GetLanguage()
	parser := sitter.NewParser()
	parser.SetLanguage(lang)

	visited := map[*types.Func]bool{}
	var results []Result
	// Global dedupe across frames/packages: file:offset:types
	seenResults := make(map[string]struct{})

	// Frame carries per-callee knowledge of which parameter names are *testing.F.
	type frame struct {
		node           *funcNode
		testingFParams map[string]bool
	}

	rootBindings := computeTestingFParamNamesFromDecl(rootDecl)
	stack := list.New()
	stack.PushBack(frame{node: rootNode, testingFParams: rootBindings})

	for stack.Len() > 0 {
		elem := stack.Back()
		stack.Remove(elem)
		fr := elem.Value.(frame)

		if visited[fr.node.obj] {
			continue
		}
		visited[fr.node.obj] = true

		src := world.srcByFile[fr.node.fname]
		tf := world.tfByFile[fr.node.fname]
		if tf == nil {
			return nil, fmt.Errorf("missing token.File for %s", fr.node.fname)
		}
		if src == nil {
			// Lazy load if not present (best-effort)
			if data, err := os.ReadFile(fr.node.fname); err == nil {
				src = data
			}
		}
		if src == nil {
			// Cannot analyze without source bytes for Tree-sitter; skip this frame.
			continue
		}

		bodyStart := tf.Offset(fr.node.decl.Body.Lbrace) + 1
		bodyEnd := tf.Offset(fr.node.decl.Body.Rbrace)

		// 1) Tree-sitter discovery
		tsCalls := treeSitterCallsInRange(parser, src, fr.node.fname, bodyStart, bodyEnd)

		// Per-body dedupe to avoid reprocessing the same call twice when both TS and AST see it.
		processed := make(map[token.Pos]struct{})

		// Process TS calls first
		for _, c := range tsCalls {
			if pos, ok := offsetToPos(world, c.file, c.start); ok {
				cexpr, _, calleeObj, sel, _, ok2 := resolveCallAtPos(world, c.file, pos)
				if !ok2 || cexpr == nil {
					continue
				}

				processed[cexpr.Lparen] = struct{}{}

				// f.Fuzz(...) detection: method named Fuzz from package "testing" or bound param of *testing.F.
				if sel != nil && sel.Sel != nil && sel.Sel.Name == "Fuzz" {
					info := world.infoByFile[c.file]
					if info != nil {
						isTestingFuzz := false

						// NEW: trust bindings — if sel.X is an Ident whose name is bound as *testing.F, accept.
						if id, ok := sel.X.(*ast.Ident); ok && fr.testingFParams != nil && fr.testingFParams[id.Name] {
							isTestingFuzz = true
						}

						// 1) Preferred: Selection -> method object
						if !isTestingFuzz {
							if selInfo := info.Selections[sel]; selInfo != nil {
								if mf, ok := selInfo.Obj().(*types.Func); ok && mf.Pkg() != nil &&
									mf.Pkg().Path() == "testing" && mf.Name() == "Fuzz" {
									isTestingFuzz = true
								}
							}
						}

						// 2) Fallback: identifier use -> method object
						if !isTestingFuzz {
							if u, ok := info.Uses[sel.Sel].(*types.Func); ok && u.Pkg() != nil &&
								u.Pkg().Path() == "testing" && u.Name() == "Fuzz" {
								isTestingFuzz = true
							}
						}

						// 3) Last resort: alias-aware receiver check
						if !isTestingFuzz {
							var recv types.Type
							if tv, ok := info.Types[sel.X]; ok && tv.Type != nil {
								recv = tv.Type
							} else if id, ok := sel.X.(*ast.Ident); ok {
								if v, ok := info.Uses[id].(*types.Var); ok && v != nil {
									recv = v.Type()
								} else if v, ok := info.Defs[id].(*types.Var); ok && v != nil {
									recv = v.Type()
								}
							}
							if recv != nil && isPtrToTestingF(recv) {
								isTestingFuzz = true
							}
						}

						if isTestingFuzz {
							var typesSlices [][]string
							for _, arg := range cexpr.Args {
								if fnLit, ok := arg.(*ast.FuncLit); ok && fnLit.Type != nil && fnLit.Type.Params != nil {
									types := extractParamTypes(fnLit.Type.Params.List)
									if len(types) > 0 {
										typesSlices = append(typesSlices, types)
									}
								}
							}
							if len(typesSlices) > 0 {
								// Global dedupe key: file:byteOffsetOfLparen:types
								tfLocal := world.tfByFile[c.file]
								var off int
								if tfLocal != nil {
									off = tfLocal.Offset(cexpr.Lparen)
								}
								flat := flatten(typesSlices)
								key := fmt.Sprintf("%s:%d:%s", c.file, off, strings.Join(flat, ","))
								if _, dup := seenResults[key]; !dup {
									seenResults[key] = struct{}{}
									fset := world.fsetByFile[c.file]
									if fset == nil {
										for _, fs := range world.fsetByFile {
											fset = fs
											break
										}
									}
									posn := fset.Position(cexpr.Lparen)
									results = append(results, Result{
										Line:  posn.Line,
										Types: flat,
									})
								}
							}
						}
					}
				}

				// Traverse into callee (cross-file / cross-package), carrying bindings
				if fnObj, ok := calleeObj.(*types.Func); ok && fnObj != nil {
					if next := findFuncNodeForObject(world, fnObj); next != nil && next.decl.Body != nil {
						bind := inferTestingFBindingsForCall(world, c.file, fr.testingFParams, cexpr, next.decl)
						stack.PushBack(frame{node: next, testingFParams: bind})
					}
				}
			}
		}

		// 2) AST fallback discovery (handles generics or grammar gaps)
		for _, ce := range enumerateASTCalls(fr.node.decl.Body) {
			if _, seen := processed[ce.Lparen]; seen {
				continue
			}
			calleeObj, sel, _ := resolveCallFromCallExpr(world, fr.node.fname, ce)

			// f.Fuzz(...) detection with testing-method preference + alias-aware fallback + bindings
			if sel != nil && sel.Sel != nil && sel.Sel.Name == "Fuzz" {
				info := world.infoByFile[fr.node.fname]
				if info != nil {
					isTestingFuzz := false

					// NEW: bindings first
					if id, ok := sel.X.(*ast.Ident); ok && fr.testingFParams != nil && fr.testingFParams[id.Name] {
						isTestingFuzz = true
					}

					// 1) Preferred: Selection -> method object
					if !isTestingFuzz {
						if selInfo := info.Selections[sel]; selInfo != nil {
							if mf, ok := selInfo.Obj().(*types.Func); ok && mf.Pkg() != nil &&
								mf.Pkg().Path() == "testing" && mf.Name() == "Fuzz" {
								isTestingFuzz = true
							}
						}
					}

					// 2) Fallback: identifier use -> method object
					if !isTestingFuzz {
						if u, ok := info.Uses[sel.Sel].(*types.Func); ok && u.Pkg() != nil &&
							u.Pkg().Path() == "testing" && u.Name() == "Fuzz" {
							isTestingFuzz = true
						}
					}

					// 3) Last resort: alias-aware receiver check
					if !isTestingFuzz {
						var recv types.Type
						if tv, ok := info.Types[sel.X]; ok && tv.Type != nil {
							recv = tv.Type
						} else if id, ok := sel.X.(*ast.Ident); ok {
							if v, ok := info.Uses[id].(*types.Var); ok && v != nil {
								recv = v.Type()
							} else if v, ok := info.Defs[id].(*types.Var); ok && v != nil {
								recv = v.Type()
							}
						}
						if recv != nil && isPtrToTestingF(recv) {
							isTestingFuzz = true
						}
					}

					if isTestingFuzz {
						var typesSlices [][]string
						for _, arg := range ce.Args {
							if fnLit, ok := arg.(*ast.FuncLit); ok && fnLit.Type != nil && fnLit.Type.Params != nil {
								types := extractParamTypes(fnLit.Type.Params.List)
								if len(types) > 0 {
									typesSlices = append(typesSlices, types)
								}
							}
						}
						if len(typesSlices) > 0 {
							tfLocal := world.tfByFile[fr.node.fname]
							var off int
							if tfLocal != nil {
								off = tfLocal.Offset(ce.Lparen)
							}
							flat := flatten(typesSlices)
							key := fmt.Sprintf("%s:%d:%s", fr.node.fname, off, strings.Join(flat, ","))
							if _, dup := seenResults[key]; !dup {
								seenResults[key] = struct{}{}
								fset := world.fsetByFile[fr.node.fname]
								if fset == nil {
									for _, fs := range world.fsetByFile {
										fset = fs
										break
									}
								}
								posn := fset.Position(ce.Lparen)
								results = append(results, Result{
									Line:  posn.Line,
									Types: flat,
								})
							}
						}
					}
				}
			}

			// Traverse into callee (carry bindings)
			if fnObj, ok := calleeObj.(*types.Func); ok && fnObj != nil {
				if next := findFuncNodeForObject(world, fnObj); next != nil && next.decl.Body != nil {
					bind := inferTestingFBindingsForCall(world, fr.node.fname, fr.testingFParams, ce, next.decl)
					stack.PushBack(frame{node: next, testingFParams: bind})
				}
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no matching f.Fuzz(func(...){...}) found inside %q", funcName)
	}
	return results, nil
}

// enumerateASTCalls returns all *ast.CallExpr nodes inside the given function body.
func enumerateASTCalls(body *ast.BlockStmt) []*ast.CallExpr {
	var out []*ast.CallExpr
	if body == nil {
		return out
	}
	ast.Inspect(body, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			out = append(out, ce)
			return true
		}
		return true
	})
	return out
}

// resolveCallFromCallExpr resolves the callee object / method info directly from a CallExpr.
// This mirrors resolveCallAtPos but works with the call node we already have (AST fallback).
func resolveCallFromCallExpr(w *astWorld, filename string, ce *ast.CallExpr) (callee types.Object, sel *ast.SelectorExpr, isMethod bool) {
	info := w.infoByFile[filename]
	if info == nil || ce == nil || ce.Fun == nil {
		return nil, nil, false
	}

	switch fun := ce.Fun.(type) {
	case *ast.SelectorExpr:
		if selInfo := info.Selections[fun]; selInfo != nil {
			return selInfo.Obj(), fun, true
		}
		if obj := info.Uses[fun.Sel]; obj != nil {
			return obj, fun, false
		}
		return nil, fun, false

	case *ast.Ident:
		if obj := info.Uses[fun]; obj != nil {
			return obj, nil, false
		}
		if obj := info.Defs[fun]; obj != nil {
			return obj, nil, false
		}
		return nil, nil, false

	case *ast.IndexExpr:
		if obj, selExpr, isMeth := resolveCalleeFromExpr(info, fun.X); obj != nil {
			return obj, selExpr, isMeth
		}
		return nil, nil, false

	case *ast.IndexListExpr:
		if obj, selExpr, isMeth := resolveCalleeFromExpr(info, fun.X); obj != nil {
			return obj, selExpr, isMeth
		}
		return nil, nil, false

	default:
		// function value or other form; we don't need to name the callee to continue traversal
		return nil, nil, false
	}
}

func findFuncDeclsByName(f *ast.File, name string) []*ast.FuncDecl {
	if f == nil {
		return nil
	}
	var out []*ast.FuncDecl
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fd.Name != nil && fd.Name.Name == name {
			out = append(out, fd)
		}
		return true
	})
	return out
}

func findFuncNodeForObject(w *astWorld, fn *types.Func) *funcNode {
	for filePath, f := range w.filesByPath {
		info := w.infoByFile[filePath]
		if info == nil {
			continue
		}
		var found *ast.FuncDecl
		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			obj, _ := info.Defs[fd.Name].(*types.Func)
			if obj == nil {
				return true
			}
			if sameFuncObject(obj, fn) {
				found = fd
				return false
			}
			return true
		})
		if found != nil {
			return &funcNode{obj: fn, decl: found, fname: filePath}
		}
	}
	return nil
}

// sameFuncObject returns true if a and b refer to the same declared function,
// even when one is an instantiation of a generic function.
// We first try pointer equality, then fall back to (pkg path, name, receiver) match.
func sameFuncObject(a, b *types.Func) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try to compare origins if available (for newer Go versions where funcs may have Origin).
	type funcWithOrigin interface{ Origin() *types.Func }
	if ao, ok := any(a).(funcWithOrigin); ok {
		if bo, ok := any(b).(funcWithOrigin); ok {
			if ao.Origin() != nil && ao.Origin() == bo.Origin() {
				return true
			}
		}
	}

	// Fallback: same package path and same name; also check receiver presence/type.
	ap, bp := pkgPathOf(a), pkgPathOf(b)
	if ap != "" && ap == bp && a.Name() == b.Name() {
		asig := a.Type().(*types.Signature)
		bsig := b.Type().(*types.Signature)
		ar, br := asig.Recv(), bsig.Recv()
		if (ar == nil) != (br == nil) {
			return false
		}
		if ar == nil {
			// Both are top-level functions.
			return true
		}
		// Compare receiver types (best-effort).
		return types.Identical(ar.Type().Underlying(), br.Type().Underlying())
	}
	return false
}

func pkgPathOf(f *types.Func) string {
	if f == nil || f.Pkg() == nil {
		return ""
	}
	return f.Pkg().Path()
}

func offsetToPos(w *astWorld, filename string, offset int) (token.Pos, bool) {
	tf := w.tfByFile[filename]
	if tf == nil {
		return token.NoPos, false
	}
	if offset < 0 || offset > tf.Size() {
		return token.NoPos, false
	}
	return tf.Pos(offset), true
}

// resolveCallAtPos finds the smallest *ast.CallExpr enclosing pos in the given file,
// and returns the call, its position, the resolved callee object (if any), and whether
// it's a method call (based on Selections).
func resolveCallAtPos(
	w *astWorld,
	filename string,
	pos token.Pos,
) (call *ast.CallExpr, callPos token.Pos, callee types.Object, sel *ast.SelectorExpr, isMethod bool, ok bool) {

	f := w.filesByPath[filename]
	info := w.infoByFile[filename]
	if f == nil || info == nil {
		return nil, 0, nil, nil, false, false
	}

	var best *ast.CallExpr
	bestDepth := 1 << 30
	curDepth := 0

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			curDepth--
			return true
		}
		curDepth++
		if ce, okCE := n.(*ast.CallExpr); okCE {
			if ce.Pos() <= pos && pos <= ce.End() && curDepth < bestDepth {
				best, bestDepth = ce, curDepth
			}
		}
		return true
	})
	if best == nil {
		return nil, 0, nil, nil, false, false
	}

	switch fun := best.Fun.(type) {
	case *ast.SelectorExpr:
		if selInfo := info.Selections[fun]; selInfo != nil {
			return best, best.Lparen, selInfo.Obj(), fun, true, true
		}
		if obj := info.Uses[fun.Sel]; obj != nil {
			return best, best.Lparen, obj, fun, false, true
		}
		return best, best.Lparen, nil, fun, false, false

	case *ast.Ident:
		if obj := info.Uses[fun]; obj != nil {
			return best, best.Lparen, obj, nil, false, true
		}
		if obj := info.Defs[fun]; obj != nil {
			return best, best.Lparen, obj, nil, false, true
		}
		return best, best.Lparen, nil, nil, false, false

	case *ast.IndexExpr:
		// Generic call with a single type arg: foo[T](...)
		if obj, selExpr, isMeth := resolveCalleeFromExpr(info, fun.X); obj != nil {
			return best, best.Lparen, obj, selExpr, isMeth, true
		}
		if t := info.Types[fun].Type; t != nil {
			return best, best.Lparen, nil, nil, false, true
		}
		return best, best.Lparen, nil, nil, false, false

	case *ast.IndexListExpr:
		// Generic call with multiple type args: foo[T1, T2](...)
		if obj, selExpr, isMeth := resolveCalleeFromExpr(info, fun.X); obj != nil {
			return best, best.Lparen, obj, selExpr, isMeth, true
		}
		if t := info.Types[fun].Type; t != nil {
			return best, best.Lparen, nil, nil, false, true
		}
		return best, best.Lparen, nil, nil, false, false

	default:
		if t := info.Types[fun].Type; t != nil {
			return best, best.Lparen, nil, nil, false, true
		}
	}
	return nil, 0, nil, nil, false, false
}

// resolveCalleeFromExpr resolves a callee object from an identifier/selector expression,
// used when the call is a generic instantiation (IndexExpr/IndexListExpr).
func resolveCalleeFromExpr(info *types.Info, e ast.Expr) (obj types.Object, sel *ast.SelectorExpr, isMethod bool) {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		if selInfo := info.Selections[x]; selInfo != nil {
			return selInfo.Obj(), x, true
		}
		if o := info.Uses[x.Sel]; o != nil {
			return o, x, false
		}
		return nil, x, false
	case *ast.Ident:
		if o := info.Uses[x]; o != nil {
			return o, nil, false
		}
		if o := info.Defs[x]; o != nil {
			return o, nil, false
		}
		return nil, nil, false
	default:
		return nil, nil, false
	}
}

/**********************
 * Support / utilities
 **********************/

// findModuleRoot walks up from start until it finds a go.mod, returning (dir, true).
// If not found, returns (start, false).
func findModuleRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start, false
		}
		dir = parent
	}
}

/******************************
 * Fuzz argument type helpers
 ******************************/

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

// isAstTestingF recognizes *testing.F from AST syntax (handles any number of leading '*').
func isAstTestingF(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.StarExpr:
		return isAstTestingF(x.X)
	case *ast.SelectorExpr:
		if x.Sel != nil && x.Sel.Name == "F" {
			if pkg, ok := x.X.(*ast.Ident); ok && pkg.Name == "testing" {
				return true
			}
		}
	}
	return false
}

// isPtrToTestingF checks whether t ultimately denotes *testing.F,
// handling aliases-to-pointers (e.g., type Fuzzer = *testing.F).
func isPtrToTestingF(t types.Type) bool {
	t = unaliasAll(t)

	// Require an outer pointer.
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}

	// Chase through aliases and accidental extra pointers coming from aliasing.
	elem := ptr.Elem()
	for {
		elem = unaliasAll(elem)
		if p, ok := elem.(*types.Pointer); ok {
			// Alias might have expanded to another pointer; peel it.
			elem = p.Elem()
			continue
		}
		break
	}

	named, ok := elem.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Name() == "F" && named.Obj().Pkg().Path() == "testing"
}

// unaliasAll peels type aliases until the type is no longer a named alias.
func unaliasAll(t types.Type) types.Type {
	for {
		if n, ok := t.(*types.Named); ok {
			tn := n.Obj() // *types.TypeName
			if tn != nil && tn.IsAlias() {
				t = n.Underlying()
				continue
			}
		}
		return t
	}
}

// computeTestingFParamNamesFromDecl returns parameter names that are directly typed as *testing.F.
func computeTestingFParamNamesFromDecl(fd *ast.FuncDecl) map[string]bool {
	out := make(map[string]bool)
	if fd == nil || fd.Type == nil || fd.Type.Params == nil {
		return out
	}
	for _, field := range fd.Type.Params.List {
		if isAstTestingF(field.Type) {
			for _, n := range field.Names {
				out[n.Name] = true
			}
		}
	}
	return out
}

// inferTestingFBindingsForCall maps callee param names to true if the corresponding argument
// at the callsite is statically a *testing.F. It also includes params directly typed as *testing.F.
func inferTestingFBindingsForCall(world *astWorld, callerFile string, callerBindings map[string]bool, call *ast.CallExpr, callee *ast.FuncDecl) map[string]bool {
	bind := computeTestingFParamNamesFromDecl(callee) // start with direct types
	paramNames := flattenParamNames(callee)
	info := world.infoByFile[callerFile]
	if info == nil {
		return bind
	}
	for i := 0; i < len(paramNames) && i < len(call.Args); i++ {
	    name := paramNames[i]
	    if name == "" {
	        continue
	    }
	    if ty := typeOfExpr(world, callerFile, call.Args[i]); ty != nil && isPtrToTestingF(ty) {
	        bind[name] = true
	        continue
	    }
	    // NEW: if the arg is an identifier bound as *testing.F in the caller frame, propagate.
	    if id, ok := call.Args[i].(*ast.Ident); ok && callerBindings != nil && callerBindings[id.Name] {
	        bind[name] = true
	    }
	}
	return bind
}

// flattenParamNames returns parameter names in order, expanding grouped params (a,b int).
func flattenParamNames(fd *ast.FuncDecl) []string {
	if fd == nil || fd.Type == nil || fd.Type.Params == nil {
		return nil
	}
	var out []string
	for _, field := range fd.Type.Params.List {
		if len(field.Names) == 0 {
			out = append(out, "")
			continue
		}
		for _, n := range field.Names {
			out = append(out, n.Name)
		}
	}
	return out
}

// typeOfExpr returns the static type of e in the given file, with fallbacks for idents.
func typeOfExpr(w *astWorld, filename string, e ast.Expr) types.Type {
	info := w.infoByFile[filename]
	if info == nil {
		return nil
	}
	if tv, ok := info.Types[e]; ok && tv.Type != nil {
		return tv.Type
	}
	if id, ok := e.(*ast.Ident); ok {
		if v, ok := info.Uses[id].(*types.Var); ok && v != nil {
			return v.Type()
		}
		if v, ok := info.Defs[id].(*types.Var); ok && v != nil {
			return v.Type()
		}
	}
	return nil
}

func exprString(e ast.Expr) string {
	// Use syntax printer (not go/types) to preserve the textual form used in tests.
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}

func flatten(x [][]string) []string {
	var out []string
	for _, inner := range x {
		out = append(out, inner...)
	}
	return out
}

/*******************************
 * JSON & seed helper functions
 *******************************/

// equalStrings is used by MergeFuncTypesIntoJSON.
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
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	return os.Chmod(path, 0o777)
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
			return nil // idempotent
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

// ConvertSeedsToGoTests:
//  1) loads types for funcName from jsonPath,
//  2) builds an empty fuzz func: func(t *testing.T, ...types) {},
//  3) for each file in seedsDir, calls Source.CreateGoTestcase with the empty func,
//  4) writes the generated Go file to outDir, named as <md5(content)>.go.
//
// Returns number of files written.
func ConvertSeedsToGoTests(seedsDir, outDir, jsonPath, funcName string) (int, error) {
	typesList, err := LoadTypesFromJSONFile(jsonPath, funcName)
	if err != nil {
		return 0, err
	}
	emptyFn, err := MakeEmptyFuzzFunc(typesList)
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

		// 2) Invoke CreateGoTestcaseWithBoundaries for better multi-parameter fuzzer support
		outVal := src.CreateGoTestcaseWithBoundaries(emptyFn.Interface(), reflect.ValueOf(new(testing.T)))

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

		if err := os.WriteFile(dst, outBytes, 0o777); err != nil {
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
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value { return nil })
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

	// Common fully spelled slice forms for clarity.
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
// if they differ, it still returns the first.
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

/*****************
 * TS utilities
 *****************/

type tsCall struct {
	file        string
	start, end  int
	calleeBytes []byte
}

func treeSitterCallsInRange(parser *sitter.Parser, src []byte, filename string, start, end int) []tsCall {
	tree := parser.Parse(nil, src)
	defer tree.Close()
	root := tree.RootNode()

	var out []tsCall
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if !n.IsNamed() {
			return
		}
		if n.Type() == "call_expression" {
			fnNode := n.ChildByFieldName("function")
			if fnNode != nil {
				s := int(fnNode.StartByte())
				e := int(fnNode.EndByte())
				if s >= start && e <= end {
					out = append(out, tsCall{
						file:        filename,
						start:       s,
						end:         e,
						calleeBytes: src[s:e],
					})
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return out
}
