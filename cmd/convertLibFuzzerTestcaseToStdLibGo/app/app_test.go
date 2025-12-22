package app

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeSource_BasicOrder(t *testing.T) {
	src := `
package p
import "testing"
func FuzzOrder(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string, f32 float32, i int, b []byte) {})
}`
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
		t.Fatalf("order mismatch: got %v want %v", got, want)
	}
}

func TestAnalyzeSource_GroupedParams(t *testing.T) {
	src := `
package p
import "testing"
func FuzzGrouped(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, a, b int, ok bool, s string) {})
}`
	res, err := AnalyzeSource([]byte(src), "FuzzGrouped")
	if err != nil {
		t.Fatalf("AnalyzeSource error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	got := res[0].Types
	want := []string{"[]byte", "int", "int", "bool", "string"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order mismatch: got %v want %v", got, want)
	}
}

func TestAnalyzeSource_RequiresTestingFReceiver(t *testing.T) {
	src := `
package p
import "testing"
type F struct{}
func FuzzX(f *testing.F) { f.Fuzz(func(t *testing.T, s string){}) }
func FuzzY(x *F) { x.Fuzz(func(t *testing.T, s string){}) }`
	_, err := AnalyzeSource([]byte(src), "FuzzY")
	if err == nil {
		t.Fatalf("expected error: no matching f.Fuzz with *testing.F receiver in FuzzY")
	}
}

func TestMergeAndLoadJSON_Basic(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "fuzz_types.json")

	// 1) First write
	a := []string{"[]byte", "int", "string"}
	if err := MergeFuncTypesIntoJSON(out, "FuncA", a); err != nil {
		t.Fatalf("merge 1: %v", err)
	}

	// 2) Idempotent write
	if err := MergeFuncTypesIntoJSON(out, "FuncA", []string{"[]byte", "int", "string"}); err != nil {
		t.Fatalf("merge 2: %v", err)
	}

	// 3) Different signature should NOT overwrite
	if err := MergeFuncTypesIntoJSON(out, "FuncA", []string{"string", "int", "[]byte"}); err != nil {
		t.Fatalf("merge 3: %v", err)
	}

	// Read back raw file, ensure JSON contains our key and order preserved
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, ok := m["FuncA"]
	if !ok {
		t.Fatalf("missing key FuncA")
	}
	if !reflect.DeepEqual(got, a) {
		t.Fatalf("order changed: got %v want %v", got, a)
	}

	// LoadTypesFromJSONFile finds same
	got2, err := LoadTypesFromJSONFile(out, "FuncA")
	if err != nil {
		t.Fatalf("LoadTypesFromJSONFile: %v", err)
	}
	if !reflect.DeepEqual(got2, a) {
		t.Fatalf("LoadTypesFromJSONFile mismatch: got %v want %v", got2, a)
	}
}

func TestLoadTypesFromJSONFile_MissingKey(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "params.json")
	if err := os.WriteFile(out, []byte(`{"Other":["int"]}`), 0o644); err != nil {
		t.Fatalf("seed json: %v", err)
	}
	if _, err := LoadTypesFromJSONFile(out, "Nope"); err == nil {
		t.Fatalf("expected error for missing function key")
	}
}

func TestMakeEmptyFuzzFunc_SignatureAndCall(t *testing.T) {
	params := []string{"string", "[]byte", "uint64", "bool"}
	fn, err := MakeEmptyFuzzFunc(params)
	if err != nil {
		t.Fatalf("MakeEmptyFuzzFunc: %v", err)
	}
	typ := fn.Type()
	if typ.NumIn() != len(params)+1 {
		t.Fatalf("NumIn=%d want %d", typ.NumIn(), len(params)+1)
	}
	if first := typ.In(0).String(); first != "*testing.T" {
		t.Fatalf("first param=%s want *testing.T", first)
	}
	// reflect string for []byte is []uint8
	want := []string{"string", "[]uint8", "uint64", "bool"}
	for i, w := range want {
		if got := typ.In(i + 1).String(); got != w {
			t.Fatalf("param #%d = %s want %s", i+1, got, w)
		}
	}

	// Call it — should be a no-op
	args := []reflect.Value{
		reflect.ValueOf(new(testing.T)),
		reflect.ValueOf("x"),
		reflect.ValueOf([]byte("y")),
		reflect.ValueOf(uint64(5)),
		reflect.ValueOf(true),
	}
	fn.Call(args)
}

func TestConvertSeedsToGoTests_TableDriven(t *testing.T) {
	tcs := []struct {
		name        string
		funcName    string
		types       []string
		seeds       [][]byte
		expectLines [][]string // expected lines *after* "go test fuzz v1" (one slice per seed file)
	}{
		{
			name:     "Int8Bounds",
			funcName: "FuzzI8",
			types:    []string{"int8"},
			seeds: [][]byte{
				{0x80}, // -128
				{0x7F}, // 127
			},
			expectLines: [][]string{
				{"int8(-128)"},
				{"int8(127)"},
			},
		},
		{
			name:     "UInt8Bounds",
			funcName: "FuzzU8",
			types:    []string{"uint8"},
			seeds: [][]byte{
				{0x00}, // 0
				{0xFF}, // 255
			},
			expectLines: [][]string{
				{"uint8(0)"},
				{"uint8(255)"},
			},
		},
		{
			name:     "StringAndBytes_CurrentBehavior",
			funcName: "FuzzStringBytes",
			types:    []string{"string", "[]byte"},
			// With the improved algorithm, first param gets 50% of bytes
			// Input: {0x03, 'A', 'B', 'C', 'D', 'E', 'F'} = 7 bytes
			// First param (string) gets 3 bytes: {0x03, 'A', 'B'}
			// Second param ([]byte) gets remaining 4 bytes: {'C', 'D', 'E', 'F'}
			seeds: [][]byte{
				{0x03, 'A', 'B', 'C', 'D', 'E', 'F'},
			},
			expectLines: [][]string{
				{`string("\x03AB")`, `[]byte("CDEF")`},
			},
		},
		{
			name:     "TwoScalars_Int8ThenUInt8",
			funcName: "FuzzI8U8",
			types:    []string{"int8", "uint8"},
			seeds: [][]byte{
				{0x80, 0xFF}, // -128, 255
				{0x7F, 0x00}, // 127, 0
			},
			expectLines: [][]string{
				{"int8(-128)", "uint8(255)"},
				{"int8(127)", "uint8(0)"},
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()

			// Write params JSON for this test
			jsonPath := filepath.Join(tmp, "params.json")
			if err := writeJSONMap(jsonPath, map[string][]string{tc.funcName: tc.types}); err != nil {
				t.Fatalf("writeJSONMap: %v", err)
			}

			// Seeds
			seedsDir := filepath.Join(tmp, "seeds")
			if err := os.MkdirAll(seedsDir, 0o755); err != nil {
				t.Fatalf("mkdir seeds: %v", err)
			}
			for i, data := range tc.seeds {
				name := fmt.Sprintf("seed_%02d.bin", i)
				if err := os.WriteFile(filepath.Join(seedsDir, name), data, 0o644); err != nil {
					t.Fatalf("write seed %s: %v", name, err)
				}
			}

			// Convert
			outDir := filepath.Join(tmp, "out")
			n, err := ConvertSeedsToGoTests(seedsDir, outDir, jsonPath, tc.funcName)
			if err != nil {
				t.Fatalf("ConvertSeedsToGoTests error: %v", err)
			}
			if n != len(tc.seeds) {
				t.Fatalf("written=%d want=%d", n, len(tc.seeds))
			}

			ents, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatalf("readdir out: %v", err)
			}
			if len(ents) != len(tc.seeds) {
				t.Fatalf("out files=%d want=%d", len(ents), len(tc.seeds))
			}

			// Track which expected value-sets we’ve matched (one per seed)
			matched := make([]bool, len(tc.expectLines))

			for _, e := range ents {
				if e.IsDir() {
					t.Fatalf("unexpected dir in out: %s", e.Name())
				}

				content, err := os.ReadFile(filepath.Join(outDir, e.Name()))
				if err != nil {
					t.Fatalf("read out file %s: %v", e.Name(), err)
				}
				if len(content) == 0 {
					t.Fatalf("empty generated file: %s", e.Name())
				}

				// filename must be md5(content)+".go"
				sum := md5.Sum(content)
				wantName := hex.EncodeToString(sum[:]) + ".go"
				if e.Name() != wantName {
					t.Fatalf("filename != md5(content).go: got %s want %s", e.Name(), wantName)
				}

				// Parse corpus: header + values
				s := strings.ReplaceAll(string(content), "\r\n", "\n")
				lines := strings.Split(s, "\n")
				// drop trailing empty line
				if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
					lines = lines[:len(lines)-1]
				}
				if len(lines) == 0 {
					t.Fatalf("no content lines in generated file %s", e.Name())
				}
				if got := strings.TrimSpace(lines[0]); got != "go test fuzz v1" {
					t.Fatalf("incorrect corpus header in %s; first line = %q", e.Name(), got)
				}
				values := make([]string, 0, len(lines)-1)
				for _, ln := range lines[1:] {
					ln = strings.TrimSpace(ln)
					if ln != "" {
						values = append(values, ln)
					}
				}
				if len(values) == 0 {
					t.Fatalf("no value lines after header in %s", e.Name())
				}

				// Try to match this file's values to one of the expected sets,
				// but if none match, report the *closest* mismatch with exact line info.
				found := false
				haveBest := false
				bestMsg := ""
				for i, exp := range tc.expectLines {
					if matched[i] {
						continue
					}
					// Compare lengths first
					if len(values) != len(exp) {
						msg := fmt.Sprintf(
							"%s: length mismatch: got %d lines after header, want %d\n  got:  %v\n  want: %v",
							e.Name(), len(values), len(exp), values, exp,
						)
						if !haveBest {
							bestMsg, haveBest = msg, true
						}
						continue
					}
					// Compare line-by-line; fail fast on first difference.
					mismatchIndex := -1
					for j := range exp {
						if values[j] != exp[j] {
							mismatchIndex = j
							break
						}
					}
					if mismatchIndex == -1 {
						// Perfect match
						matched[i] = true
						found = true
						break
					} else {
						lineNo := mismatchIndex + 2 // +1 for 1-based, +1 to account for header line
						msg := fmt.Sprintf(
							"%s: mismatch at line %d (after header):\n  got:  %q\n  want: %q\n  full got:  %v\n  full want: %v",
							e.Name(), lineNo, values[mismatchIndex], exp[mismatchIndex], values, exp,
						)
						if !haveBest {
							bestMsg, haveBest = msg, true
						}
					}
				}
				if !found {
					if haveBest {
						t.Fatalf("no expected value-set matched:\n%s", bestMsg)
					} else {
						t.Fatalf("no expected value-set matched %s; got lines: %v", e.Name(), values)
					}
				}
			}

			// Ensure every expected output was matched
			for i, ok := range matched {
				if !ok {
					t.Fatalf("expected output %d (%v) not found among generated files", i, tc.expectLines[i])
				}
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
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

func TestConvertSeedsToGoTests_MissingFuncKey(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "params.json")
	if err := writeJSONMap(jsonPath, map[string][]string{"Other": {"int"}}); err != nil {
		t.Fatalf("writeJSONMap: %v", err)
	}
	seedsDir := filepath.Join(dir, "seeds")
	_ = os.MkdirAll(seedsDir, 0o755)
	outDir := filepath.Join(dir, "out")

	if _, err := ConvertSeedsToGoTests(seedsDir, outDir, jsonPath, "NoSuchFunc"); err == nil {
		t.Fatalf("expected error for missing function key")
	}
}