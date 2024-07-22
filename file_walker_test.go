package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/docker/docker/daemon/graphdriver/copy"
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
	walker := NewFileWalker()
	walker.rewriteTestingFFunctionParams(file)
	gotFileContents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotFileContents, []byte(expectedFileContents)) {
		t.Errorf("%s", cmp.Diff(gotFileContents, []byte(expectedFileContents)))
	}

	if len(walker.rewrittenFiles) != 1 {
		t.Errorf("Should only have rewritten one file")
	}

	if !stringInSlice(file, walker.rewrittenFiles) {
		t.Errorf("The rewritten file %s should be stored in the walker but is not.", file)
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

func TestRenameAllTestFiles(t *testing.T) {
	tempDir := t.TempDir()
	err := copy.DirCopy(filepath.Join("testdata", "module1"),
						filepath.Join(tempDir, "module1"),
						copy.Content,
						false)
	if err != nil {
		t.Fatal(err)
	}
	originalFuzzTestPath := filepath.Join(tempDir, "module1", "fuzz_test.go")
	originalSubmodule1OnePath := filepath.Join(tempDir, "module1", "submodule1", "one.go")
	originalSubmodule1OneTestPath := filepath.Join(tempDir, "module1", "submodule1", "one_test.go")
	originalSubmodule2TestOnePath := filepath.Join(tempDir, "module1", "submodule2", "test_one.go")
	renamedFuzzTestPath := filepath.Join(tempDir, "module1", "fuzz_libFuzzer.go")
	renamedSubmodule1OneTestPath := filepath.Join(tempDir, "module1", "submodule1", "one_libFuzzer.go")
	files, err := GetAllSourceFilesOfFile(originalFuzzTestPath)
	if err != nil {
		t.Fatalf("failed to load packages: %s", err)
	}

	// Check that our test files exist before we move them
	if !fileExists(originalFuzzTestPath) {
		t.Fatal("File does not exist")
	}
	if !fileExists(originalSubmodule1OnePath) {
		t.Fatal("File does not exist")
	}
	if !fileExists(originalSubmodule1OneTestPath) {
		t.Fatal("File does not exist")
	}
	if !fileExists(originalSubmodule2TestOnePath) {
		t.Fatal("File does not exist")
	}
	walker := NewFileWalker()
	walker.RewriteAllImportedTestFiles(files)

	// Check that we rewrote the right files
	if !fileExists(renamedFuzzTestPath) {
		t.Fatal("File does not exist")
	}
	if !fileExists(originalSubmodule1OnePath) {
		t.Fatal("File does not exist")
	}
	if !fileExists(renamedSubmodule1OneTestPath) {
		t.Fatal("File does not exist")
	}
	if !fileExists(originalSubmodule2TestOnePath) {
		t.Fatal("File does not exist")
	}
	
	// We should have renamed two files
	if len(walker.renamedFiles) != 2 {
		t.Error("There should be two rewrites")
	}

	if fuzzTest, ok := walker.renamedFiles[renamedFuzzTestPath]; ok {
		if fuzzTest != renamedFuzzTestPath {
			t.Errorf("Path is %s but should be %s", fuzzTest, renamedFuzzTestPath)
		}
	}

	if oneTest, ok := walker.renamedFiles[renamedSubmodule1OneTestPath]; ok {
		if oneTest != renamedSubmodule1OneTestPath {
			t.Errorf("Path is %s but should be %s", oneTest, renamedSubmodule1OneTestPath)
		}
	}

	// Restore the files we renamed
	err = walker.RestoreRenamedTestFiles()
	if err != nil {
		t.Fatal(err)
	}

	if !fileExists(originalFuzzTestPath) {
		t.Error("File does not exist")
	}

	if !fileExists(originalSubmodule1OnePath) {
		t.Error("File does not exist")
	}

	if !fileExists(originalSubmodule1OneTestPath) {
		t.Error("File does not exist")
	}

	if !fileExists(originalSubmodule2TestOnePath) {
		t.Error("File does not exist")
	}

	// Make sure that the filenames that we changed the _test.go files
	// to DO NOT exist anymore
	if fileExists(renamedFuzzTestPath) {
		t.Fatal("File should not exist")
	}
	if fileExists(renamedSubmodule1OneTestPath) {
		t.Fatal("File should not exist")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
