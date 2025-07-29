package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"os"
	"path/filepath"
	"testing"


	"github.com/google/go-cmp/cmp"
)

/*func TestGetAllPackagesOfFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	walker.
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
}*/

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
	expectedCoverOut string
}

/*func TestCoverageFileContents(t *testing.T) {
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
}*/

func TTTestCompileCoverageFile(t *testing.T) {
	//fmt.Println(os.Getwd())
	tests := []*CoverageFileTest{
		&CoverageFileTest{
			module: "module1",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module1",
			expectedCoverageOutput: fmt.Sprintf("b is:  AA\nPASS\ncoverage: 87.5%% of statements in module1/...\n"),
			expectedCoverOut: `mode: set
module1/fuzz_libFuzzer.go:13.30,14.41 1 1
module1/fuzz_libFuzzer.go:14.41,15.20 1 1
module1/fuzz_libFuzzer.go:15.20,17.4 1 0
module1/fuzz_libFuzzer.go:18.3,18.29 1 1
module1/fuzz_libFuzzer.go:18.29,19.30 1 1
module1/fuzz_libFuzzer.go:19.30,20.31 1 1
module1/fuzz_libFuzzer.go:20.31,22.6 1 1
module1/submodule2/one.go:3.17,5.2 1 1
`,
		},
		&CoverageFileTest{
			module: "module2",
			fuzzerPath: "fuzz_test.go",
			flagFunc: "FuzzTest",
			fuzzerPackageName: "module2",
			expectedCoverageOutput: fmt.Sprintf("PASS\ncoverage: 88.9%% of statements in module2/...\n"),
			expectedCoverOut: `mode: set
module2/fuzz_libFuzzer.go:10.30,11.41 1 1
module2/fuzz_libFuzzer.go:11.41,13.20 2 1
module2/fuzz_libFuzzer.go:13.20,15.4 1 0
module2/submodule3/one.go:3.37,4.17 1 1
module2/submodule3/one.go:4.17,6.3 1 1
module2/submodule3/one.go:6.8,6.25 1 1
module2/submodule3/one.go:6.25,8.3 1 1
module2/submodule3/one.go:8.8,10.3 1 1
`,
		},
		// This is a good test but it breaks because of an upstream issue in golang: https://github.com/golang/go/issues/73802
		/*&CoverageFileTest{
			module: "module3",
			fuzzerPath: "submodule1/fuzz_test.go",
			flagFunc: "FuzzProcessItem",
			fuzzerPackageName: "submodule1",
			expectedCoverageOutput: fmt.Sprintf("b is:  AA\nPASS\ncoverage: 100.0%% of statements in module3/...\n"),
			expectedCoverOut: "mode: set\nmodule1/submodule2/one.go:3.17,5.2 1 1\n",
		},*/
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
			fuzzerPath := filepath.Join("testdata", tc.module, tc.fuzzerPath)
			absFuzzerPath := filepath.Join(pwd, fuzzerPath)
			err = os.Chdir(filepath.Dir(absFuzzerPath))
			if err != nil {
				t.Fatal(err)
			}

			walker := NewFileWalker()
			walker.sanitizer="coverage"
			defer walker.cleanUp()
			walker.fuzzerPath = absFuzzerPath // We should/could use getAbsPathOfFuzzFile here
			fmt.Println("running ", tc.module)
			walker.CreateAndModifyFiles(tc.module, tc.flagFunc, "", tc.fuzzerPackageName)
			
			// This one is probably specific to this test.
			tidyArgs := []string{"mod", "tidy"}
			tidyArgs = append(tidyArgs, walker.overlayArgs...)
			cmd := exec.Command("go", tidyArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Error(err)
			}

			// This one could be standardized
			outPath := fmt.Sprintf("./compiled_fuzzer")
			err = buildTestBinary(outPath, fmt.Sprintf("%s/...", tc.module), walker.overlayArgs)
			if err != nil {
				t.Fatal(err)
			}
			

			// Run the built coverage binary
			// Here we have to set up the seeds dir and moves the seeds
			// into it. It then sets the environment variable to the
			// path of the just created seeds dir
			modulePath := filepath.Join(pwd, "testdata", tc.module)
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
			// We have now created the seeds dir

			var outb bytes.Buffer
			coverDir := t.TempDir()
			cmd = exec.Command(outPath, "-test.run", "TestFuzzCorpus", 
				fmt.Sprintf("-test.coverprofile=%s", filepath.Join(coverDir, "cover.out")))
			cmd.Stdout = &outb
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Error(err)
			}
			// Assert the output from running the binary
			if outb.String() != tc.expectedCoverageOutput {
				fmt.Printf("outb: '%s'\n and expected output: '%s'\n", outb.String(), tc.expectedCoverageOutput)
				t.Error("Not equal")
			}
			// Assert the contents of the generated "cover.out"
			coverOutContents, err := os.ReadFile(filepath.Join(coverDir, "cover.out"))
			if err != nil {
				t.Fatal(err)
			}
			if string(coverOutContents) != tc.expectedCoverOut {
				t.Errorf("Expected '%s'\nGot '%s'\n", tc.expectedCoverOut, string(coverOutContents))
			}
		})
		continue
	}
}

// 1:
// Test that ensures that fuzzer is removed during coverage build

func TestHookPlacement(t *testing.T) {
	input := `func (c *common) Cleanup(f func()) {
}	

func (c *common) Context() context.Context {
}

func (t *T) Deadline() (deadline time.Time, ok bool) {
}
func (c *common) Error(args ...any) {
}`
	expected := `func (c *common) Cleanup(f func()) {
f()
}	

func (c *common) Context() context.Context {
return context.Background()
}

func (t *T) Deadline() (deadline time.Time, ok bool) {
panic("t.Deadline()")
}
func (c *common) Error(args ...any) {
fmt.Println(args...)
	panic("error")
}

func (t *T) TempDir() string {
	tmpFuzzDir, err := os.MkdirTemp(t.tempDirsParentDir, "fuzzdir-")
	if err != nil {
		panic(err)
	}
	return tmpFuzzDir
}`
	got := PlaceHooks(input)
	if expected != got {
		t.Errorf("%s", cmp.Diff(got, expected))
		//panic(fmt.Sprintf("got: \n%s\n\nexpected: \n%s\n\n", got, expected))
	}
}