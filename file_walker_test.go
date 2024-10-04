package main

import (
	//"bytes"
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

// TODOs:
// 1: Find a good way to prepend an "F" to the fuzz func
// 2: Find a good way to use the current go-118-fuzz-build
func TestCompileCoverageFile(t *testing.T) {
	//fmt.Println(os.Getwd())
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

			/*originalFuzzerContents, err := os.ReadFile(absFuzzerPath)
			if err != nil {
				t.Fatal(err)
			}*/


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
			walker.overlayMap.Replace["oss_fuzz_coverage_test.go"] = tempFile
			overlayJson, err := json.Marshal(walker.overlayMap)
			if err != nil {
				t.Fatal(err)
			}
			//fmt.Println("overlayJson: ", string(overlayJson))
			//t.Error("Just because")
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
			//tempFuzzContents, err := os.ReadFile(walker.overlayMap.Replace["/tmp/go-118-fuzz-build/testdata/module1/coverage_fuzzer_renamed.go"])
			//fmt.Println(string(tempFuzzContents))
			//os.Chdir(walker.overlayMap.Replace["oss_fuzz_coverage_test.go"])

			// remove fuzzer. For some reason it is giving us problems in the coverage build
			/*fuzzerCopy, err := os.CreateTemp("", "fuzzerCopy")
			if err != nil {
				t.Fatal(err)
			}*/
			/*defer func() {
				sf, err := os.Create(absFuzzerPath)
				if err != nil {
					t.Fatal(err)
				}
				_, err = sf.Write(originalFuzzerContents)
				if err != nil {
					sf.Close()
					t.Fatal(err)
				}
				err = sf.Close()
				if err != nil {
					t.Fatal(err)
				}
			}()*/

			// The fuzz function cannot be called "Fuzz*". Prefix an "F"
			/*updatedFuzzerContents := strings.Replace(string(originalFuzzerContents),
													fmt.Sprintf("func %s(", tc.flagFunc),
													fmt.Sprintf("func %s(", funcName),
													1)
			fmt.Println(string(updatedFuzzerContents))

			if _, err := fuzzerCopy.Write([]byte(updatedFuzzerContents)); err != nil {
				fuzzerCopy.Close()
				t.Fatal(err)
			}
			fuzzerCopy.Close()*/
			// Coverage doesn't work when fuzz_test.go is still there
			os.Remove(absFuzzerPath)

			cmd := exec.Command("go", "mod", "tidy", "-overlay", overlayFile.Name())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Error(err)
			}

			outPath := fmt.Sprintf("./compiled_fuzzer")
			args := []string{"test",
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
			cmd = exec.Command(outPath, "-test.run", "TestFuzzCorpus")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				t.Error(err)
			}
			os.Remove(outPath)
		})
		continue

		//newOverlayMap := &Overlay{Replace: make(map[string]string)}

		/*oldFuzzerContents, err := os.ReadFile(fuzzerPath)
		if err != nil {
			t.Fatal(err)
		}

		updatedFuzzerContents := strings.Replace(string(oldFuzzerContents), "\"testing\"", "\"github.com/AdamKorcz/go-118-fuzz-build/testing\"", 1)
		// The fuzz function cannot be called "Fuzz*". Prefix an "F"
		updatedFuzzerContents = strings.Replace(updatedFuzzerContents, tc.flagFunc, funcName, 1)
		
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
		
		coverageFilePath, tempFile, err := createCoverageRunner(fuzzerPath, funcName, tc.fuzzerPackageName)
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
		os.Chdir(pwd)*/
	}
}

// 1:
// Test that ensures that fuzzer is removed during coverage build