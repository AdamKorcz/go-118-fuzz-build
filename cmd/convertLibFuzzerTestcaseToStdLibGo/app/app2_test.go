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