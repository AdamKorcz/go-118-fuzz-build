package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

//
// Test 1: cross-package callgraph traversal
//
// Module layout:
//   example.com/mymod/p1: FuzzTest1(f) { p2.CallFuzz(f) }
//   example.com/mymod/p2: CallFuzz(f)  { f.Fuzz(func(t *testing.T, ...){}) }
//
// Expect AnalyzeFile to walk into p2.CallFuzz and extract the arg types.
//
// NOTE: With the current AnalyzeFile (single-package), this will fail until
// we upgrade it to load deps and traverse cross-package calls.
//
func TestAnalyzeFile_CrossPackageCallgraph(t *testing.T) {
	tmp := t.TempDir()

	// go.mod
	goMod := `module example.com/mymod

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Package p1 (root fuzz entry)
	p1dir := filepath.Join(tmp, "p1")
	if err := os.MkdirAll(p1dir, 0o755); err != nil {
		t.Fatalf("mkdir p1: %v", err)
	}
	p1src := `package p1

import (
	"testing"
	"example.com/mymod/p2"
)

func FuzzTest1(f *testing.F) {
	p2.CallFuzz(f)
}
`
	p1file := filepath.Join(p1dir, "fuzz_test1.go")
	if err := os.WriteFile(p1file, []byte(p1src), 0o644); err != nil {
		t.Fatalf("write p1 file: %v", err)
	}

	// Package p2 (helper that actually calls f.Fuzz)
	p2dir := filepath.Join(tmp, "p2")
	if err := os.MkdirAll(p2dir, 0o755); err != nil {
		t.Fatalf("mkdir p2: %v", err)
	}
	p2src := `package p2

import "testing"

func CallFuzz(f *testing.F) {
	f.Fuzz(func(t *testing.T, arg1 string, arg2 []byte, arg3 float32, arg4 int32){})
}
`
	p2file := filepath.Join(p2dir, "fuzz_helper.go")
	if err := os.WriteFile(p2file, []byte(p2src), 0o644); err != nil {
		t.Fatalf("write p2 file: %v", err)
	}
	_ = p2file // file is present for module completeness

	// Analyze the p1 file / FuzzTest1.
	res, err := AnalyzeFile(p1file, "FuzzTest1")
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	got := res[0].Types
	want := []string{"string", "[]byte", "float32", "int32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("type order mismatch: got %v want %v", got, want)
	}
}

//
// Test 2: deep callchain with generics and primitives only
//
// Shape (simplified after the Istio example):
//   FuzzTop(f *testing.F) -> fuzzDeepCopyEqual[*Service](f)
//   fuzzDeepCopyEqual[T equalCopier[T]](f *testing.F) -> shim(f)
//   shim(f *testing.F) -> f.Fuzz(func(t *testing.T, <primitives>){})
//
// Ensures we still extract the primitives list correctly.
//
func TestAnalyzeSource_GenericDeepChain_Primitives(t *testing.T) {
	src := `
package p

import "testing"

// A generic constraint similar in spirit to equal/copier constraints.
type equalCopier[T any] interface {
	Copy() T
}

type Service struct{ ID int }
func (s *Service) Copy() *Service { return s }

// FuzzTop passes *testing.F down through a generic helper to a shim
// that finally calls f.Fuzz with ONLY primitive fuzzable types.
func FuzzTop(f *testing.F) {
	fuzzDeepCopyEqual[*Service](f)
}

func fuzzDeepCopyEqual[T equalCopier[T]](f *testing.F) {
	// Do some unrelated generic-y work, then forward to a shim.
	var _ T
	shim(f)
}

func shim(f *testing.F) {
	f.Fuzz(func(t *testing.T, a string, b []byte, c float32, d int32){})
}
`
	res, err := AnalyzeSource([]byte(src), "FuzzTop")
	if err != nil {
		t.Fatalf("AnalyzeSource error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	got := res[0].Types
	want := []string{"string", "[]byte", "float32", "int32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("type order mismatch: got %v want %v", got, want)
	}
}

func TestAnalyzeSource_GenericDeepChain_Branching(t *testing.T) {
	src := `
package p

import "testing"

// Generic constraint, similar to earlier tests
type equalCopier[T any] interface {
	Copy() T
}

type ServiceA struct{ ID int }
func (s *ServiceA) Copy() *ServiceA { return s }

type ServiceB struct{ ID int }
func (s *ServiceB) Copy() *ServiceB { return s }

// Two fuzz shims with different parameter lists
func shim(f *testing.F) {
	f.Fuzz(func(t *testing.T, a string, b []byte){})
}
func shim2(f *testing.F) {
	f.Fuzz(func(t *testing.T, x int32, y float32){})
}

// Root fuzz entries select the same generic helper with different type args
func FuzzTopA(f *testing.F) { fuzzDeepCopyEqual[*ServiceA](f) }
func FuzzTopB(f *testing.F) { fuzzDeepCopyEqual[*ServiceB](f) }

// Generic helper that branches based on T's concrete type.
func fuzzDeepCopyEqual[T equalCopier[T]](f *testing.F) {
	var z T
	switch any(z).(type) {
	case *ServiceA:
		shim(f)
	case *ServiceB:
		shim2(f)
	default:
		shim(f)
	}
}
`

	containsTypes := func(results []Result, want []string) bool {
		for _, r := range results {
			if reflect.DeepEqual(r.Types, want) {
				return true
			}
		}
		return false
	}

	// Case A: expect ["string","[]byte"]
	resA, err := AnalyzeSource([]byte(src), "FuzzTopA")
	if err != nil {
		t.Fatalf("AnalyzeSource(FuzzTopA) error: %v", err)
	}
	wantA := []string{"string", "[]byte"}
	if !containsTypes(resA, wantA) {
		t.Fatalf("expected to find %v among results; got %v", wantA, resA)
	}

	// Case B: expect ["int32","float32"]
	resB, err := AnalyzeSource([]byte(src), "FuzzTopB")
	if err != nil {
		t.Fatalf("AnalyzeSource(FuzzTopB) error: %v", err)
	}
	wantB := []string{"int32", "float32"}
	if !containsTypes(resB, wantB) {
		t.Fatalf("expected to find %v among results; got %v", wantB, resB)
	}
}

// This test simulates the Istio-style helper pattern:
//   - Package "fuzz": Fuzz(f Fuzzer, ff func(Helper)) { f.Fuzz(func(*testing.T, data []byte){...}) }
//   - Package "p2":   FuzzCreateCertificate(f *testing.F) { fuzz.Fuzz(f, func(fg fuzz.Helper){ ... }) }
//
// We expect AnalyzeFile on FuzzCreateCertificate to traverse into fuzz.Fuzz and
// extract the inner []byte parameter from the f.Fuzz(func(...)) call.
func TestAnalyzeFile_HelperPackage_ExtractBytesParam(t *testing.T) {
	tmp := t.TempDir()

	// go.mod
	goMod := `module example.com/mymod

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Package fuzz (the helper)
	fuzzDir := filepath.Join(tmp, "fuzz")
	if err := os.MkdirAll(fuzzDir, 0o755); err != nil {
		t.Fatalf("mkdir fuzz: %v", err)
	}
	fuzzSrc := `package fuzz

import "testing"

// Helper and utilities to mirror the shape of Istio's helper.
type Helper struct{}

type Fuzzer interface {
	Fuzz(any)
}

func BaseCases(f Fuzzer) {}
func Finalize()          {}
func New(t *testing.T, data []byte) Helper { return Helper{} }

// Fuzz is the helper that ultimately calls (*testing.F).Fuzz with a func literal taking []byte.
func Fuzz(f Fuzzer, ff func(Helper)) {
	BaseCases(f)
	// Ensure the static type at the callsite is *testing.F so analysis recognizes the receiver.
	if tf, ok := f.(*testing.F); ok {
		tf.Fuzz(func(t *testing.T, data []byte) {
			defer Finalize()
			fg := New(t, data)
			ff(fg)
		})
	}
}
`
	fuzzFile := filepath.Join(fuzzDir, "util.go")
	if err := os.WriteFile(fuzzFile, []byte(fuzzSrc), 0o644); err != nil {
		t.Fatalf("write fuzz util.go: %v", err)
	}

	// Package p2 (the test target)
	p2Dir := filepath.Join(tmp, "p2")
	if err := os.MkdirAll(p2Dir, 0o755); err != nil {
		t.Fatalf("mkdir p2: %v", err)
	}
	p2Src := `package p2

import (
	"testing"
	"example.com/mymod/fuzz"
)

// FuzzCreateCertificate calls into the helper in another package.
func FuzzCreateCertificate(f *testing.F) {
	fuzz.Fuzz(f, func(fg fuzz.Helper) {
		// The body isn't relevant for analysis; the inner f.Fuzz(...) is in the helper.
	})
}
`
	p2File := filepath.Join(p2Dir, "fuzz_create_cert_test.go")
	if err := os.WriteFile(p2File, []byte(p2Src), 0o644); err != nil {
		t.Fatalf("write p2 file: %v", err)
	}

	// Analyze the p2 file / FuzzCreateCertificate.
	res, err := AnalyzeFile(p2File, "FuzzCreateCertificate")
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d (res=%v)", len(res), res)
	}
	got := res[0].Types
	want := []string{"[]byte"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("types mismatch: got %v want %v", got, want)
	}
}

// Mirrors the shape:
//
// package ca
//
// import (
//   "testing"
//   "istio.io/istio/pkg/fuzz"
// )
//
// func FuzzCreateCertificate(f *testing.F) {
//   fuzz.Fuzz(f, func(fg fuzz.Helper) {
//     _ = 0
//   })
// }
//
// where pkg fuzz defines:
//
// func Fuzz(f test.Fuzzer, ff func(fg Helper)) {
//   BaseCases(f)
//   f.Fuzz(func(t *testing.T, data []byte) {
//     defer Finalize()
//     fg := New(t, data)
//     ff(fg)
//   })
// }
//
// and test.Fuzzer comes from istio.io/istio/pkg/fuzz/test.
//
// We expect to extract exactly []string{"[]byte"} from the inner f.Fuzz(func(...)).
func TestAnalyzeFile_IstioHelper_ExactSignature_ExtractBytesParam(t *testing.T) {
	tmp := t.TempDir()

	// Module root: istio.io/istio
	goMod := `module istio.io/istio

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Subpkg: istio.io/istio/pkg/fuzz/test — define test.Fuzzer as an interface.
	testDir := filepath.Join(tmp, "pkg", "fuzz", "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("mkdir fuzz/test: %v", err)
	}
	testSrc := `package test

// Fuzzer abstracts *testing.F
type Fuzzer interface {
	Fuzz(ff any)
	Add(args ...any)
}
`
	if err := os.WriteFile(filepath.Join(testDir, "fuzzer.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatalf("write fuzz/test/fuzzer.go: %v", err)
	}

	// Package: istio.io/istio/pkg/fuzz — helper with the EXACT Fuzz signature/body you want.
	fuzzDir := filepath.Join(tmp, "pkg", "fuzz")
	if err := os.MkdirAll(fuzzDir, 0o755); err != nil {
		t.Fatalf("mkdir pkg/fuzz: %v", err)
	}
	fuzzSrc := `package fuzz

import (
	"testing"
	test "istio.io/istio/pkg/fuzz/test"
)

type Helper struct{}

func BaseCases(f test.Fuzzer) {}
func Finalize()               {}
func New(t *testing.T, data []byte) Helper { return Helper{} }

// EXACT helper signature/body as requested.
func Fuzz(f test.Fuzzer, ff func(fg Helper)) {
	BaseCases(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		defer Finalize()
		fg := New(t, data)
		ff(fg)
	})
}
`
	if err := os.WriteFile(filepath.Join(fuzzDir, "util.go"), []byte(fuzzSrc), 0o644); err != nil {
		t.Fatalf("write pkg/fuzz/util.go: %v", err)
	}

	// Package: istio.io/istio/ca — the target function that calls into the helper.
	caDir := filepath.Join(tmp, "ca")
	if err := os.MkdirAll(caDir, 0o755); err != nil {
		t.Fatalf("mkdir ca: %v", err)
	}
	caSrc := `package ca

import (
	"testing"
	"istio.io/istio/pkg/fuzz"
)

func FuzzCreateCertificate(f *testing.F) {
	fuzz.Fuzz(f, func(fg fuzz.Helper) {
		_ = 0
	})
}
`
	caFile := filepath.Join(caDir, "fuzz_ca_test.go")
	if err := os.WriteFile(caFile, []byte(caSrc), 0o644); err != nil {
		t.Fatalf("write ca file: %v", err)
	}

	// Analyze and assert
	res, err := AnalyzeFile(caFile, "FuzzCreateCertificate")
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d (res=%v)", len(res), res)
	}
	got := res[0].Types
	want := []string{"[]byte"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("types mismatch: got %v want %v", got, want)
	}
}

func TestAnalyzeFile_DeepCopyChain_InterfacePropagation(t *testing.T) {
	tmp := t.TempDir()

	// Module root
	goMod := `module istio.io/istio

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// pkg/fuzz/test: Fuzzer interface
	testDir := filepath.Join(tmp, "pkg", "fuzz", "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("mkdir fuzz/test: %v", err)
	}
	testSrc := `package test

// Fuzzer abstracts *testing.F
type Fuzzer interface {
	Fuzz(ff any)
	Add(args ...any)
}
`
	if err := os.WriteFile(filepath.Join(testDir, "fuzzer.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatalf("write fuzz/test/fuzzer.go: %v", err)
	}

	// pkg/fuzz: helper + stubs used by the deepcopy code
	fuzzDir := filepath.Join(tmp, "pkg", "fuzz")
	if err := os.MkdirAll(fuzzDir, 0o755); err != nil {
		t.Fatalf("mkdir pkg/fuzz: %v", err)
	}
	fuzzSrc := `package fuzz

import "testing"
import test "istio.io/istio/pkg/fuzz/test"

type Helper struct{}

func BaseCases(f test.Fuzzer) {}
func Finalize()               {}
func New(t *testing.T, data []byte) Helper { return Helper{} }

// Minimal helpers used by the deepcopy pipeline
func (Helper) T() *testing.T { return new(testing.T) }

func Struct[T any](Helper) T          { var zero T; return zero }
func DeepCopySlow[T any](v T) T       { return v }
func MutateStruct(_ *testing.T, _ any) {}

// EXACT helper that ultimately calls (*testing.F).Fuzz
func Fuzz(f test.Fuzzer, ff func(fg Helper)) {
	BaseCases(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		defer Finalize()
		fg := New(t, data)
		ff(fg)
	})
}
`
	if err := os.WriteFile(filepath.Join(fuzzDir, "util.go"), []byte(fuzzSrc), 0o644); err != nil {
		t.Fatalf("write pkg/fuzz/util.go: %v", err)
	}

	// pkg/assert: no-op Equal to satisfy imports in the sample
	assertDir := filepath.Join(tmp, "pkg", "assert")
	if err := os.MkdirAll(assertDir, 0o755); err != nil {
		t.Fatalf("mkdir pkg/assert: %v", err)
	}
	assertSrc := `package assert

import "testing"

func Equal(_ *testing.T, _ any, _ any) {}
`
	if err := os.WriteFile(filepath.Join(assertDir, "assert.go"), []byte(assertSrc), 0o644); err != nil {
		t.Fatalf("write pkg/assert/assert.go: %v", err)
	}

	// ca package: root fuzz + generic hop + interface param
	caDir := filepath.Join(tmp, "ca")
	if err := os.MkdirAll(caDir, 0o755); err != nil {
		t.Fatalf("mkdir ca: %v", err)
	}
	caSrc := `package ca

import (
	"testing"
	"istio.io/istio/pkg/assert"
	"istio.io/istio/pkg/fuzz"
	test "istio.io/istio/pkg/fuzz/test"
)

type ServiceInstance struct{}

func (s *ServiceInstance) DeepCopy() *ServiceInstance { return s }

type deepCopier[T any] interface {
	DeepCopy() T
}

func FuzzDeepCopyServiceInstance(f *testing.F) {
	fuzzDeepCopy[*ServiceInstance](f)
}

func fuzzDeepCopy[T deepCopier[T]](f test.Fuzzer) {
	fuzz.Fuzz(f, func(fg fuzz.Helper) {
		orig := fuzz.Struct[T](fg)
		fast := orig.DeepCopy()
		slow := fuzz.DeepCopySlow[T](orig)

		// check copy is correct
		assert.Equal(fg.T(), orig, fast)
		assert.Equal(fg.T(), orig, slow)

		// check is deep copy
		fuzz.MutateStruct(fg.T(), &orig)
		assert.Equal(fg.T(), fast, slow)
	})
}
`
	caFile := filepath.Join(caDir, "fuzz_deepcopy_test.go")
	if err := os.WriteFile(caFile, []byte(caSrc), 0o644); err != nil {
		t.Fatalf("write ca file: %v", err)
	}

	// Analyze and assert
	res, err := AnalyzeFile(caFile, "FuzzDeepCopyServiceInstance")
	if err != nil {
		t.Fatalf("AnalyzeFile error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d (res=%v)", len(res), res)
	}
	got := res[0].Types
	want := []string{"[]byte"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("types mismatch: got %v want %v", got, want)
	}

	// cilium case
	statedbDir := filepath.Join(tmp, "pkg", "statedb")
	if err := os.MkdirAll(statedbDir, 0o755); err != nil {
		t.Fatalf("mkdir pkg/statedb: %v", err)
	}
	statedbSrc := `package statedb

// TableWritable is intentionally empty here; any type satisfies it for our test.
type TableWritable interface{}
`
	if err := os.WriteFile(filepath.Join(statedbDir, "statedb.go"), []byte(statedbSrc), 0o644); err != nil {
		t.Fatalf("write pkg/statedb/statedb.go: %v", err)
	}

	// loadbalancer package with BOTH functions in the same file, as requested.
	lbDir := filepath.Join(tmp, "pkg", "loadbalancer")
	if err := os.MkdirAll(lbDir, 0o755); err != nil {
		t.Fatalf("mkdir pkg/loadbalancer: %v", err)
	}
	lbSrc := `package loadbalancer

import (
	"testing"
	"cilium/pkg/statedb"
)

// Backend type to use as the concrete type argument.
type Backend struct{}

// Entry point that calls the generic helper with a concrete type argument.
func FuzzJSONBackend(f *testing.F) {
	tableRowJSONFuzzer[*Backend](f)
}

// Generic helper (same file) that ultimately calls (*testing.F).Fuzz with []byte.
func tableRowJSONFuzzer[T statedb.TableWritable](f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		// body not relevant for analysis
	})
}
`
	lbFile := filepath.Join(lbDir, "fuzz_jsonbackend_test.go")
	if err := os.WriteFile(lbFile, []byte(lbSrc), 0o644); err != nil {
		t.Fatalf("write pkg/loadbalancer/fuzz_jsonbackend_test.go: %v", err)
	}

	// Ensure the package is discoverable by packages.Load by adding one non-test file.
	lbStub := `package loadbalancer

	// stub ensures this package has at least one non-test source file.
	const _loadbalancerStub = 0
	`
	if err := os.WriteFile(filepath.Join(lbDir, "zz_stub.go"), []byte(lbStub), 0o644); err != nil {
		t.Fatalf("write pkg/loadbalancer/zz_stub.go: %v", err)
	}

	cacheMu.Lock()
	delete(moduleWorldCache, tmp) // tmp is the absolute module root path
	cacheMu.Unlock()

	// Analyze and assert for the Cilium-style case.
	res3, err := AnalyzeFile(lbFile, "FuzzJSONBackend")
	if err != nil {
		t.Fatalf("AnalyzeFile(FuzzJSONBackend) error: %v", err)
	}
	if len(res3) != 1 {
		t.Fatalf("expected 1 result for FuzzJSONBackend, got %d (res=%v)", len(res3), res3)
	}
	got3 := res3[0].Types
	want3 := []string{"[]byte"}
	if !reflect.DeepEqual(got3, want3) {
		t.Fatalf("FuzzJSONBackend types mismatch: got %v want %v", got3, want3)
	}
}
