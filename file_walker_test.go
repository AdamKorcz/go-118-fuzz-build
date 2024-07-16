package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRewriteFuncTestingFParams(t *testing.T) {
	fileContents := `package main
import (
	"testing"
)
func ourTestHelper(f *testing.F) {
	_ = f
}`
	expectedFileContents := `package main

import (
	"testing"
)

func ourTestHelper(f *customFuzzTestingPkg.F) {
	_ = f
}
`
	file := filepath.Join(t.TempDir(), "file.go")
	err := os.WriteFile(file, []byte(fileContents), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	rewriteTestingFFunctionParams(file)
	gotFileContents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotFileContents, []byte(expectedFileContents)) {
		t.Errorf("%s", cmp.Diff(gotFileContents, []byte(expectedFileContents)))
	}
}

func TestGetAllPackagesOfFile(t *testing.T) {
	pkgs, err := getAllPackagesOfFile(filepath.Join("testdata", "module1", "fuzz_test.go"))
	if err != nil {
		t.Fatalf("failed to load packages: %s", err)
	}
	if pkgs[0].Name != "module1" {
		t.Error("pkgs[0].Name should be 'module1'")
	}
	if pkgs[1].Name != "submodule1" {
		t.Error("pkgs[1].Name should be 'submodule1'")
	}
	if pkgs[2].Name != "submodule2" {
		t.Error("pkgs[2].Name should be 'submodule2'")
	}
	if pkgs[3].Name != "submodule1_test" {
		t.Error("pkgs[3].Name should be 'submodule1_test'")
	}
	if pkgs[4].Name != "main" {
		t.Error("pkgs[4].Name should be 'main'")
	}
}

func TestGetAllSourceFilesOfFile(t *testing.T) {
	files, err := GetAllSourceFilesOfFile(filepath.Join("testdata", "module1", "fuzz_test.go"))
	if err != nil {
		t.Fatalf("failed to load packages: %s", err)
	}
	if filepath.Base(files[0]) != "fuzz_test.go" {
		t.Error("files[0] should be 'fuzz_test.go'")
	}
	if filepath.Base(files[1]) != "one.go" {
		t.Error("files[1] should be 'one.go'")
	}
	if filepath.Base(files[2]) != "test_one.go" {
		t.Error("files[2] should be 'test_one.go'")
	}
	if filepath.Base(files[3]) != "one_test.go" {
		t.Error("files[3] should be 'one_test.go'")
	}
}
