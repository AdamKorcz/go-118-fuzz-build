package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAnalyzeSource_PreservesOrder(t *testing.T) {
	src := `
package p

import "testing"

func FuzzOrder(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string, f32 float32, i int, b []byte) {})
}
`
	res, err := AnalyzeSource([]byte(src), "FuzzOrder")
	if err != nil {
		t.Fatalf("AnalyzeSource error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	got := res[0].Types
	want := []string{"string", "float32", "int", "[]byte"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestAnalyzeSource_PreservesOrder_WithGroupedParams(t *testing.T) {
	src := `
package p

import "testing"

func FuzzGroupedOrder(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, a, b int, ok bool, s string) {})
}
`
	res, err := AnalyzeSource([]byte(src), "FuzzGroupedOrder")
	if err != nil {
		t.Fatalf("AnalyzeSource error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	got := res[0].Types
	want := []string{"[]byte", "int", "int", "bool", "string"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestAnalyzeSource_StrictReceiver(t *testing.T) {
	src := `
package p

import "testing"

type F struct{} // not testing.F

func FuzzX(f *testing.F) {
	// OK: receiver is *testing.F
	f.Fuzz(func(t *testing.T, s string) {})
}

func FuzzY(x *F) {
	// NOT OK: receiver is not *testing.F, should be ignored
	x.Fuzz(func(t *testing.T, s string) {})
}
`
	_, err := AnalyzeSource([]byte(src), "FuzzY")
	if err == nil {
		t.Fatalf("expected error for FuzzY (no matching f.Fuzz inside), got nil")
	}
}

func TestAnalyzeSource_NoTestingTFirstParam(t *testing.T) {
	src := `
package p

import "testing"

func FuzzWeird(f *testing.F) {
	// Even if the inner func doesn't start with *testing.T, we don't skip anything.
	f.Fuzz(func(ctx int, s string) {})
}
`
	res, err := AnalyzeSource([]byte(src), "FuzzWeird")
	if err != nil {
		t.Fatalf("AnalyzeSource error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	got := res[0].Types
	want := []string{"int", "string"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("types/order mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestJSONOrder_SimpleWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "fuzz_types.json")

	want := []string{"string", "float32", "int", "[]byte"}
	if err := MergeFuncTypesIntoJSON(out, "FuncTwo", want); err != nil {
		t.Fatalf("merge error: %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v\njson: %s", err, string(raw))
	}

	got, ok := m["FuncTwo"]
	if !ok {
		t.Fatalf("missing key FuncTwo")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order mismatch in JSON array:\n got  %v\n want %v", got, want)
	}
}

func TestJSONOrder_EndToEnd_FromAnalyzeSource(t *testing.T) {
	src := []byte(`
package p
import "testing"
func FuzzGrouped(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, a, b int, ok bool, s string) {})
}
`)
	results, err := AnalyzeSource(src, "FuzzGrouped")
	if err != nil {
		t.Fatalf("AnalyzeSource error: %v", err)
	}
	types := ChooseTypes(results, true)

	dir := t.TempDir()
	out := filepath.Join(dir, "fuzz_types.json")
	if err := MergeFuncTypesIntoJSON(out, "FuzzGrouped", types); err != nil {
		t.Fatalf("merge error: %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v\njson: %s", err, string(raw))
	}

	got := m["FuzzGrouped"]
	want := []string{"[]byte", "int", "int", "bool", "string"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order mismatch in JSON array:\n got  %v\n want %v", got, want)
	}
}

func TestJSONOrder_NotOverwrittenAndPreservedOnRewrites(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "fuzz_types.json")

	orig := []string{"[]byte", "int", "string"} // specific order
	if err := MergeFuncTypesIntoJSON(out, "FuzzA", orig); err != nil {
		t.Fatalf("merge 1: %v", err)
	}

	// Re-merge with identical types: should be a no-op, order unchanged.
	if err := MergeFuncTypesIntoJSON(out, "FuzzA", []string{"[]byte", "int", "string"}); err != nil {
		t.Fatalf("merge 2: %v", err)
	}

	// Attempt to merge with a different signature: per requirements, do not overwrite.
	if err := MergeFuncTypesIntoJSON(out, "FuzzA", []string{"string", "int", "[]byte"}); err != nil {
		t.Fatalf("merge 3: %v", err)
	}

	// Read back and verify original order is intact.
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v\njson: %s", err, string(raw))
	}
	got := m["FuzzA"]
	if !reflect.DeepEqual(got, orig) {
		t.Fatalf("order changed unexpectedly:\n got  %v\n want %v", got, orig)
	}
}