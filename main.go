package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

type Data struct {
	PkgPath      string
	Func         string
	Declarations string
	FuzzerParams string
	PkgName      string
}

var (
	flagFunc      = flag.String("func", "Fuzz", "fuzzer entry point")
	flagO         = flag.String("o", "", "output file")
	flagPath      = flag.String("abs_path", "", "absolute path to fuzzer")
	flagSanitizer = flag.String("sanitizer", "address", "The sanitizer to compile the target with. Either 'address' or 'coverage'")

	flagRace    = flag.Bool("race", false, "enable data race detection")
	flagTags    = flag.String("tags", "", "a comma-separated list of build tags to consider satisfied during the build")
	flagV       = flag.Bool("v", false, "print the names of packages as they are compiled")
	flagWork    = flag.Bool("work", false, "print the name of the temporary work directory and do not remove it when exiting")
	flagX       = flag.Bool("x", false, "print the commands")
	flagOverlay = flag.String("overlay", "", "JSON config file that provides an overlay for build operations")

	flagInclude  = flag.String("include", "*", "a comma-separated list of import paths to instrument")
	flagPreserve = flag.String("preserve", "", "a comma-separated list of import paths not to instrument")

	LoadMode = packages.NeedName |
		packages.NeedFiles |
		packages.NeedCompiledGoFiles |
		packages.NeedImports |
		packages.NeedDeps |
		packages.NeedTypes |
		packages.NeedModule
)

var include, ignore []string

func main() {
	flag.Parse()

	if !token.IsIdentifier(*flagFunc) || !token.IsExported(*flagFunc) {
		log.Fatal("-func must be an exported identifier")
	}

	tags := "gofuzz_libfuzzer,libfuzzer"
	if *flagTags != "" {
		tags += "," + *flagTags
	}

	buildFlags := []string{
		"-buildmode", "c-archive",
		"-tags", tags,
		"-trimpath",
	}

	if *flagRace {
		buildFlags = append(buildFlags, "-race")
	}
	if *flagV {
		buildFlags = append(buildFlags, "-v")
	}
	if *flagWork {
		buildFlags = append(buildFlags, "-work")
	}
	if *flagX {
		buildFlags = append(buildFlags, "-x")
	}

	if len(flag.Args()) != 1 {
		log.Fatal("must specify exactly one package path")
	}
	path := flag.Args()[0]
	if strings.Contains(path, "...") {
		log.Fatal("package path must not contain ... wildcards")
	}
	sanitizer := *flagSanitizer

	include = strings.Split(*flagInclude, ",")
	ignore = []string{
		"runtime/cgo",   // No reason to instrument these.
		"runtime/pprof", // No reason to instrument these.
		"runtime/race",  // No reason to instrument these.
		"syscall",       // https://github.com/google/oss-fuzz/issues/3639
	}
	if *flagPreserve != "" {
		ignore = append(ignore, strings.Split(*flagPreserve, ",")...)
	}
	buildFlags = append(buildFlags, "-gcflags", "all=-d=libfuzzer")

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	//fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags,
		Tests:      true,
	}, "pattern="+path)
	if err != nil {
		log.Fatal("failed to load packages:", err)
	}
	fuzzerPackage := pkgs[0]
	var modulePath string
	if fuzzerPackage.Module == nil {
		modulePath = ""
	} else {
		modulePath = fuzzerPackage.Module.Path
	}
	visit := func(pkg *packages.Package) {
		if !shouldInstrument(pkg.PkgPath) {
			buildFlags = append(buildFlags, "-gcflags", pkg.PkgPath+"=-d=libfuzzer=0")
		}
	}
	packages.Visit(pkgs, nil, visit)
	if packages.PrintErrors(pkgs) != 0 {
		os.Exit(1)
	}
	fuzzerPath, err := getAbsPathOfFuzzFile(fuzzerPackage.PkgPath, *flagFunc, buildFlags)
	if err != nil {
		panic(err)
	}
	allFiles, err := GetAllSourceFilesOfFile(modulePath, fuzzerPath)
	if err != nil {
		panic(err)
	}
	walker := NewFileWalker()
	defer walker.cleanUp()
	for _, sourceFile := range allFiles {
		walker.RewriteFile(sourceFile, fuzzerPath, *flagFunc)
	}
	err = os.Chdir(cwd)
	if err != nil {
		panic(err)
	}

	overlayArgs := make([]string, 0)

	// Merge overlay maps
	newOverlayMap := &Overlay{Replace: make(map[string]string)}
	if *flagOverlay != "" {
		b, err := os.ReadFile(*flagOverlay)
		if err != nil {
			panic(fmt.Sprintf("Could not find overlay file %s", err.Error()))
		}
		usersOverlayMap := &Overlay{}
		err = json.Unmarshal(b, usersOverlayMap)
		if err != nil {
			panic(fmt.Sprintf("Could not read overlay file %s", err.Error()))
		}
		for k, v := range usersOverlayMap.Replace {
			newOverlayMap.Replace[k] = v
		}
	}
	for k, v := range walker.overlayMap.Replace {
		newOverlayMap.Replace[k] = v
	}
	if sanitizer == "coverage" {
		coverageFilePath, tempFile, err := createCoverageRunner(fuzzerPath, *flagFunc, fuzzerPackage.Name)
		if err != nil {
			panic(err)
		}
		newOverlayMap.Replace[coverageFilePath] = tempFile
		defer os.Remove(tempFile)
	}
	if len(newOverlayMap.Replace) > 0 {
		overlayFile, err := os.CreateTemp("", "ossFuzzOverlayFile.json")
		if err != nil {
			panic(err)
		}
		overlayJson, err := json.Marshal(newOverlayMap)
		if err != nil {
			panic(err)
		}
		defer os.Remove(overlayFile.Name())
		if _, err := overlayFile.Write(overlayJson); err != nil {
			overlayFile.Close()
			panic(err)
		}
		overlayFile.Close()
		fmt.Println("overlayJson: ", string(overlayJson))
		overlayArgs = append(overlayArgs, "-overlay", overlayFile.Name())
	}

	if sanitizer == "address" {
		importPath := fuzzerPackage.PkgPath
		if strings.HasPrefix(importPath, "_/") {
			importPath = path
		}

		mainFile, err := ioutil.TempFile(".", "main.*.go")
		if err != nil {
			log.Fatal("failed to create temporary file:", err)
		}
		defer func() {
			fmt.Println("removing mainFile.Name()")
			err = os.Remove(mainFile.Name())
			if err != nil {
				panic(err)
			}
		}()
		err = mainTmpl.Execute(mainFile, &Data{
			PkgPath: importPath,
			Func:    *flagFunc,
		})
		if err != nil {
			log.Fatal("failed to execute template:", err)
		}
		if err := mainFile.Close(); err != nil {
			log.Fatal(err)
		}

		out := *flagO
		if out == "" {
			out = fuzzerPackage.Name + "-fuzz.a"
		}

		args := []string{"build", "-o", out}
		args = append(args, buildFlags...)
		if len(overlayArgs) > 0 {
			args = append(args, overlayArgs...)
		}
		args = append(args, mainFile.Name())
		fmt.Println("Running go ", args)
		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			panic(err)
			log.Fatal("failed to build packages:", err)
		}
	} else {
		// coverage sanitizer
		// 3. Compile it with: https://github.com/google/oss-fuzz/blob/690b4ebae2e7dcf69bde5bfcbf4c668f8f177ca0/infra/base-images/base-builder/compile_go_fuzzer#L60

		outPath := fmt.Sprintf("%s/%s", os.Getenv("OUT"), *flagO)

		args := []string{"test", "-run", "TestFuzzCorpus",
			"-vet=off"}
		if len(overlayArgs) > 0 {
			args = append(args, overlayArgs...)
		}

		args = append(args, "-coverpkg", "./...",
			"-c", "-o", outPath)
		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			panic(err)
			log.Fatal("failed to build packages:", err)
		}

		// 4. Clean up test filen.

	}
	fmt.Println("BUILT THE FUZZER")
}

// Returns the path to the coverage test and the temp file. The user should add
// this to the overlay map with "coverageFilePath":f.Name()"
func createCoverageRunner(fuzzerPath, flagFunc, fuzzerPackageName string) (string, string, error) {
	pathForCoverageTest := filepath.Dir(fuzzerPath)
	coverageFilePath := filepath.Join(pathForCoverageTest, "oss_fuzz_coverage_test.go")
	f, err := os.CreateTemp("", "coverageFile")
	if err != nil {
		return coverageFilePath, "", err
	}
	defer f.Close()
	err = coverageTmpl.Execute(f, &Data{
		Func:    flagFunc,
		PkgName: fuzzerPackageName,
	})
	return coverageFilePath, f.Name(), nil
}

// Packages that match one of the include patterns (default is include all packages)
// and none of the exclude patterns (default is none) will be instrumented.
func shouldInstrument(pkgPath string) bool {
	for _, incPath := range include {
		if matchPattern(incPath, pkgPath) {
			for _, excPath := range ignore {
				if matchPattern(excPath, pkgPath) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func matchPattern(pattern, path string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return strings.EqualFold(path, pattern)
}

var mainTmpl = template.Must(template.New("main").Parse(`
// Code generated by go-118-fuzz-build; DO NOT EDIT.

// +build ignore

package main

import (
	"runtime"
	"strings"
	"unsafe"
	target {{printf "%q" .PkgPath}}
	"github.com/AdamKorcz/go-118-fuzz-build/testing"
)

// #include <stdint.h>
import "C"

//export LLVMFuzzerTestOneInput
func LLVMFuzzerTestOneInput(data *C.char, size C.size_t) C.int {
	s := (*[1<<30]byte)(unsafe.Pointer(data))[:size:size]
	defer catchPanics()
	LibFuzzer{{.Func}}(s)
	return 0
}

func LibFuzzer{{.Func}}(data []byte) int {
	fuzzer := testing.NewF(data)
	defer fuzzer.CleanupTempDirs()
	target.{{.Func}}(fuzzer)
	return 1
}

func catchPanics() {
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
}

func main() {
}
`))

var coverageTmpl = template.Must(template.New("fuzz_coverage_report_test").Parse(`

package {{.PkgName}}

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
		data, err := os.ReadFile(fname)
		if err != nil {
			t.Error("Failed to read corpus file", err)
			return err
		}
		fuzzer := customTesting.NewF(data)
		defer func(){
			fuzzer.CleanupTempDirs()
		}()
		{{.Func}}(fuzzer)
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
`))
