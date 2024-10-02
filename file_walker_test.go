package main

import (
	//"bytes"
	"fmt"
	"os/exec"
	"os"
	"path/filepath"
	"strings"
	"encoding/json"
	"testing"

	//"github.com/google/go-cmp/cmp"
	//"github.com/docker/docker/daemon/graphdriver/copy"
)

func TestGetAllPackagesOfFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := getAllPackagesOfFile("module1", filepath.Join("testdata", "module1", "fuzz_test.go"))
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
	os.Chdir(pwd)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var expectedCoverageFiles = map[string]string {
	"module1": `

package module1

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	customTesting "github.com/AdamKorcz/go-118-fuzz-build/testing"
)

func TestFuzzCorpus(t *testing.T) {
	dir := os.Getenv("FUZZ_CORPUS_DIR")
	if dir == "" {
		t.Logf("No fuzzing corpus directory set")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			var err string
			switch r.(type) {
			case string:
				err = r.(string)
			case runtime.Error:
				err = r.(runtime.Error).Error()
			case error:
				err = r.(error).Error()
			}
			if strings.Contains(err, "GO-FUZZ-BUILD-PANIC") {
				return
			} else {
				panic(err)
			}
		}
	}()
	profname := os.Getenv("FUZZ_PROFILE_NAME")
	if profname != "" {
		f, err := os.Create(profname + ".cpu.prof")
		if err != nil {
			t.Logf("error creating profile file %s\n", err)
		} else {
			_ = pprof.StartCPUProfile(f)
		}
	}
	_, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Logf("Not fuzzing corpus directory %s", err)
		return
	}
	// recurse for regressions subdirectory
	err = filepath.Walk(dir, func(fname string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		data, err := ioutil.ReadFile(fname)
		if err != nil {
			t.Error("Failed to read corpus file", err)
			return err
		}
		fuzzer := testing.NewF(data)
		defer func(){
			fuzzer.CleanupTempDirs()
		}()
		FuzzTest(fuzzer)
		return nil
	})
	if err != nil {
		t.Error("Failed to run corpus", err)
	}
	if profname != "" {
		pprof.StopCPUProfile()
		f, err := os.Create(profname + ".heap.prof")
		if err != nil {
			t.Logf("error creating heap profile file %s\n", err)
		}
		if err = pprof.WriteHeapProfile(f); err != nil {
			t.Logf("error writing heap profile file %s\n", err)
		}
		f.Close()
	}
}
`,
}

type CoverageFileTest struct {
	module string
	fuzzerPath string // relative to ./testdata/module
	flagFunc string
	fuzzerPackageName string
	expectedFilePath string
}
func TestCoverageFilePath(t *testing.T) {
	tests := []*CoverageFileTest{
		&CoverageFileTest{
			module: "module1",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module1",
			expectedFilePath: "module1/oss_fuzz_coverage_test.go",
		},

	}
	for _, test := range tests {
		fuzzerPath := filepath.Join("testdata", test.module, test.fuzzerPath)
		coverageFilePath, tempFile, err := createCoverageRunner(fuzzerPath, test.flagFunc, test.fuzzerPackageName)
		if err != nil {
			t.Error(err)
		}
		expectedPath := filepath.Join("testdata", test.expectedFilePath)
		if expectedPath != coverageFilePath {
			os.Remove(tempFile)
			t.Errorf("Expected %s but got %s", expectedPath, coverageFilePath)
		}
		os.Remove(coverageFilePath)
	}
}

func TestCoverageFileContents(t *testing.T) {
	tests := []*CoverageFileTest{
		&CoverageFileTest{
			module: "module1",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module1",
		},

	}
	for _, test := range tests {
		fuzzerPath := filepath.Join("testdata", test.module, test.fuzzerPath)
		_, tempFile, err := createCoverageRunner(fuzzerPath, test.flagFunc, test.fuzzerPackageName)
		if err != nil {
			t.Error(err)
		}
		gotFileContents, err := os.ReadFile(tempFile)
		if err != nil {			
			os.Remove(tempFile)
			t.Error(err)
		}
		if string(gotFileContents) != expectedCoverageFiles[test.module] {
			t.Errorf("Did not create the correct file contents. \n Got: %s\n\nExpected: %s", string(gotFileContents), expectedCoverageFiles[test.module])
		}
		os.Remove(tempFile)
	}
}

type Overlay struct {
	Replace map[string]string
}

func TestCompileCoverageFile(t *testing.T) {
	fmt.Println(os.Getwd())
	tests := []*CoverageFileTest{
		&CoverageFileTest{
			module: "module1",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module1",
		},
		&CoverageFileTest{
			module: "module2",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module2",
		},
	}
	fmt.Println(os.Getwd())
	for _, test := range tests {
		pwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		funcName := fmt.Sprintf("F%s", test.flagFunc)

		fuzzerPath := filepath.Join("testdata", test.module, test.fuzzerPath)
		absFuzzerPath := filepath.Join(pwd, fuzzerPath)
		// Rename "testing" to "github.com/AdamKorcz/go-118-fuzz-build/testing".
		// This is not the best way to do it, but it is enough to get started
		// with a single or a few tests. Ideally we should use our librarys
		// utilities to do this.
		oldFuzzerContents, err := os.ReadFile(fuzzerPath)
		if err != nil {
			t.Fatal(err)
		}
		updatedFuzzerContents := strings.Replace(string(oldFuzzerContents), "\"testing\"", "\"github.com/AdamKorcz/go-118-fuzz-build/testing\"", 1)
		// The fuzz function cannot be called "Fuzz*". Prefix an "F"
		updatedFuzzerContents = strings.Replace(updatedFuzzerContents, test.flagFunc, funcName, 1)
		
		tempFuzzer, err := os.CreateTemp("", "temp_fuzzer.go")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tempFuzzer.Name())
		if _, err := tempFuzzer.Write([]byte(updatedFuzzerContents)); err != nil {
			tempFuzzer.Close()
			t.Fatal(err)
		}
		tempFuzzer.Close()
		newPath := fmt.Sprintf("%s_libfuzzer.go", absFuzzerPath)
		err = os.Rename(absFuzzerPath, newPath)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Rename(newPath, absFuzzerPath)
		
		coverageFilePath, tempFile, err := createCoverageRunner(fuzzerPath, funcName, test.fuzzerPackageName)
		if err != nil {
			t.Error(err)
		}
		defer os.Remove(tempFile)
		overlayMap := &Overlay{
			Replace: map[string]string {
				coverageFilePath: tempFile,
			},
		}
		overlayJson, err := json.Marshal(overlayMap)
		if err != nil {
			t.Fatal(err)
		}
		overlayFile, err := os.CreateTemp("", "overlay.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(overlayFile.Name())
		if _, err := overlayFile.Write(overlayJson); err != nil {
			overlayFile.Close()
			t.Fatal(err)
		}
		overlayFile.Close()


		outPath := fmt.Sprintf("./compiled_fuzzer")

		fmt.Println("cd to ", filepath.Join(filepath.Dir(coverageFilePath)))
		err = os.Chdir(filepath.Join(filepath.Dir(coverageFilePath)))
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			cwd, err := os.Getwd()
			if err != nil {
				t.Error(err)
			}
			if cwd != pwd {
				os.Chdir(pwd)
			}
		}()
		cmd := exec.Command("go", "mod", "tidy", "-overlay", overlayFile.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
		args := []string{"test", "-overlay", overlayFile.Name(),
			"-vet=off", // otherwise vet will complain unnecessarily
			"-run", "TestFuzzCorpus",
			"-c", "-o", outPath}
		cmd = exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			t.Error(err)
		}
		os.Remove(outPath)
		os.Chdir(pwd)
	}
}