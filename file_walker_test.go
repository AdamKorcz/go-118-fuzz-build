package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"os"
	"path/filepath"
	//"strings"
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
		fuzzer := customTesting.NewF(data)
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
	expectedCoverageOutput string
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

func TestCompileCoverageFile(t *testing.T) {
	//fmt.Println(os.Getwd())
	tests := []*CoverageFileTest{
		&CoverageFileTest{
			module: "module1",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module1",
			expectedCoverageOutput: fmt.Sprintf("b is:  A\nPASS\n"),
		},
		&CoverageFileTest{
			module: "module2",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module2",
			expectedCoverageOutput: fmt.Sprintf("b is:  b\nPASS\n"),
		},
	}
	//fmt.Println(os.Getwd())
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.module, func(t *testing.T) {
			pwd, err := os.Getwd()
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
			funcName := fmt.Sprintf("F%s", tc.flagFunc)
			modulePath := filepath.Join(pwd, "testdata", tc.module)
			fuzzerPath := filepath.Join("testdata", tc.module, tc.fuzzerPath)
			absFuzzerPath := filepath.Join(pwd, fuzzerPath)
			err = os.Chdir(filepath.Dir(absFuzzerPath))
			if err != nil {
				t.Fatal(err)
			}

			allFiles, err := GetAllSourceFilesOfFile(tc.module, absFuzzerPath)
			if err != nil {
				t.Fatal(err)
			}
			walker := NewFileWalker()
			walker.sanitizer="coverage"
			defer walker.cleanUp()
			for _, sourceFile := range allFiles {
				walker.RewriteFile(sourceFile, absFuzzerPath, tc.flagFunc)
			}
			// Here we could assert the contents of the overlaymap
		
			// Create coverage runner
			coverageFilePath, tempFile, err := createCoverageRunner(fuzzerPath, funcName, tc.fuzzerPackageName)
			if err != nil {
				t.Error(err)
			}
			_ = coverageFilePath
			defer os.Remove(tempFile)
			// Could make the key here absolute:
			walker.overlayMap.Replace["oss_fuzz_coverage_test.go"] = tempFile
			overlayJson, err := json.Marshal(walker.overlayMap)
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
			

			cmd := exec.Command("go", "mod", "tidy", "-overlay", overlayFile.Name())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Error(err)
			}

			outPath := fmt.Sprintf("./compiled_fuzzer")
			args := []string{"test",
				"-covermode=atomic",
				"-overlay", overlayFile.Name(),
				"-vet=off", // otherwise vet will complain unnecessarily
				"-c", "-o", outPath, "-v"}
			cmd = exec.Command("go", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Error(err)
			}

			// Run the built coverage binary
			corpusDir := t.TempDir()
			seedFiles, err := os.ReadDir(filepath.Join(modulePath, "seeds"))
			if err != nil {
				t.Fatal(err)
			}
			for _, seedFile := range seedFiles {
				seedFileContent, err := os.ReadFile(filepath.Join(filepath.Join(modulePath, "seeds"), seedFile.Name()))
				if err != nil {
					t.Fatal(err)
				}
				sf, err := os.Create(filepath.Join(corpusDir, seedFile.Name()))
				if err != nil {
					t.Fatal(err)
				}
				sf.Write(seedFileContent)
				sf.Close()
			}
			os.Setenv("FUZZ_CORPUS_DIR", corpusDir)
			var outb bytes.Buffer
			cmd = exec.Command(outPath, "-test.run", "TestFuzzCorpus", "-test.coverprofile=cover.out")
			cmd.Stdout = &outb
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Error(err)
			}
			if outb.String() != tc.expectedCoverageOutput {
				t.Error("Not equal")
			}
			bbbbb, err := os.ReadFile("cover.out")
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(string(bbbbb))
			t.Error("Just because")
			os.Remove(outPath)
		})
		continue
	}
}

// 1:
// Test that ensures that fuzzer is removed during coverage build