package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	customTestingName = "customFuzzTestingPkg"

	buildFlags2 = []string{
		"-buildmode", "c-archive",
		"-trimpath",
		"-gcflags", "all=-d=libfuzzer",
	}
)

type Overlay struct {
	Replace map[string]string
}

type FileWalker struct {
	renamedFiles     map[string]string
	renamedTestFiles map[string]string // key = old, correct name, value = temporary name
	rewrittenFiles   []string
	// Stores the original files
	originalFiles map[string]string
	tmpDir        string
	overlayMap    *Overlay
	sanitizer     string
	fuzzerPath    string
	allFiles      []string
	overlayArgs   []string
}

func NewFileWalker() *FileWalker {
	tmpDir, err := os.MkdirTemp("", "gofuzzbuild")
	if err != nil {
		panic(err)
	}
	return &FileWalker{
		renamedFiles:     make(map[string]string),
		renamedTestFiles: make(map[string]string),
		rewrittenFiles:   make([]string, 0),
		originalFiles:    make(map[string]string),
		tmpDir:           tmpDir,
		overlayMap:       &Overlay{Replace: make(map[string]string)},
		allFiles:         make([]string, 0),
		overlayArgs:      make([]string, 0),
	}
}

func (walker *FileWalker) cleanUp() {
	for oldName, renamedTestFile := range walker.renamedTestFiles {
		err := os.Rename(renamedTestFile, oldName)
		if err != nil {
			panic(err)
		}
	}
	// Remove the visible fuzzer path
	//if walker.sanitizer == "coverage" {
	os.Remove(strings.TrimSuffix(walker.fuzzerPath, "_test.go") + "_libFuzzer.go")
	//}
	/*for _, renamedTestFile := range walker.renamedTestFiles {
		fmt.Println("Cleaning up1... ", renamedTestFile)
		newName := strings.TrimSuffix(renamedTestFile, "_libFuzzer.go") + "_test.go"
		err := os.Rename(renamedTestFile, oldName)
		if err != nil {
			panic(err)
		}
	}*/
	err := os.RemoveAll(walker.tmpDir)
	if err != nil {
		panic(err)
	}
}

func (walker *FileWalker) createRewrittenHarness(path string, fset1 *token.FileSet, parsedFile *ast.File) error {
	fmt.Println("creating rewritten harness")
	originalFuzzerContents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	originalFuzzerFileCopy, err := os.CreateTemp(walker.tmpDir, "")
	if err != nil {
		return err
	}
	_, err = originalFuzzerFileCopy.Write(originalFuzzerContents)
	if err != nil {
		return err
	}
	if err = originalFuzzerFileCopy.Close(); err != nil {
		return err
	}
	visibleFuzzerPath := strings.TrimSuffix(walker.fuzzerPath, "_test.go") + "_libFuzzer.go"
	fff, err := os.Create(visibleFuzzerPath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	printer.Fprint(&buf, fset1, parsedFile)

	_, err = fff.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if err = fff.Close(); err != nil {
		return err
	}

	walker.renamedTestFiles[walker.fuzzerPath] = originalFuzzerFileCopy.Name()
	err = os.Remove(path)
	if err != nil {
		return err
	}
	fmt.Println("created rewritten harness")
	return nil
}

// "path" is expected to be a file in a module
// that a fuzzer uses.
func (walker *FileWalker) RewriteFile(path, fuzzFuncName string) {
	if filepath.Ext(path) != ".go" {
		return
	}

	//fileName := filepath.Base(path)
	if strings.HasSuffix(path, "_test.go") {
		if filepath.Dir(path) != filepath.Dir(walker.fuzzerPath) {
			return
		}
	}

	// TODO: Check if it is a "_test" pkg outside of the fuzzers dir.
	// If it is, then we should not rewrite it.
	fset1 := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset1, path, nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	// If coverage: prepend "F"
	if walker.sanitizer == "coverage" && strings.EqualFold(path, walker.fuzzerPath) {

		// Change fuzz function name from Fuzz* to FFuzz*
		for _, decl := range parsedFile.Decls {
			if _, ok := decl.(*ast.FuncDecl); ok {
				if decl.(*ast.FuncDecl).Name.Name == fuzzFuncName {
					decl.(*ast.FuncDecl).Name.Name = fmt.Sprintf("F%s", fuzzFuncName)
				}
			}
		}

		// Make a copy of the original fuzzer contents
		err = walker.createRewrittenHarness(path, fset1, parsedFile)
		if err != nil {
			panic(err)
		}
	} else if path[len(path)-8:] == "_test.go" && filepath.Dir(path) == filepath.Dir(walker.fuzzerPath) {
		fmt.Println("renaming _test.go file in fuzzer dir: ", path)
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return
		}
		f, err := os.CreateTemp(walker.tmpDir, "")
		if err != nil {
			panic(err)
		}

		_, err = f.Write(fileBytes)
		if err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
		keyName := strings.TrimSuffix(path, "_test.go") + "_libFuzzer.go"
		walker.overlayMap.Replace[keyName] = f.Name()
	}
}

// Gets the full path of the file in which the "func Fuzz" is
func (walker *FileWalker) getAbsPathOfFuzzFile(pkgPath, fuzzerName string, buildFlags []string) error {
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags,
		Tests:      true,
	}, "pattern="+pkgPath)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		if pkg.PkgPath != pkgPath {
			continue
		}
		for _, file := range pkg.GoFiles {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, 0)
			if err != nil {
				return err
			}
			for _, decl := range f.Decls {
				if _, ok := decl.(*ast.FuncDecl); ok {
					if decl.(*ast.FuncDecl).Name.Name == fuzzerName {
						walker.fuzzerPath = file
						return nil

					}
				}
			}
		}
	}
	return fmt.Errorf("Could not find the fuzz func")
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (walker *FileWalker) CreateOverlayFile(usersOverlayFile string) []string {
	overlayArgs := make([]string, 0)
	// Merge overlay maps
	newOverlayMap := &Overlay{Replace: make(map[string]string)}
	if usersOverlayFile != "" {
		b, err := os.ReadFile(usersOverlayFile)
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
	if len(newOverlayMap.Replace) > 0 {
		overlayFile, err := os.CreateTemp(walker.tmpDir, "ossFuzzOverlayFile.json")
		if err != nil {
			panic(err)
		}
		overlayJson, err := json.Marshal(newOverlayMap)
		if err != nil {
			panic(err)
		}
		if _, err := overlayFile.Write(overlayJson); err != nil {
			overlayFile.Close()
			panic(err)
		}
		overlayFile.Close()
		overlayArgs = append(overlayArgs, "-overlay", overlayFile.Name())
	}
	return overlayArgs
}

// Returns the path to the coverage test and the temp file. The user should add
// this to the overlay map with "coverageFilePath":f.Name()"
func (walker *FileWalker) createCoverageRunner(flagFunc, fuzzerPackageName string) error {
	modifiedFuncName := fmt.Sprintf("F%s", flagFunc)
	f, err := os.CreateTemp(walker.tmpDir, "coverageFile")
	if err != nil {
		return err
	}
	defer f.Close()
	err = coverageTmpl.Execute(f, &Data{
		Func:    modifiedFuncName,
		PkgName: fuzzerPackageName,
	})
	walker.overlayMap.Replace["oss_fuzz_coverage_test.go"] = f.Name()
	return nil
}

func (walker *FileWalker) CreateAndModifyFiles(modulePath, fuzzerFuncName, flagOverlay, fuzzerPackage string) {
	if walker.sanitizer == "coverage" {
		walker.createCoverageRunner(fuzzerFuncName, fuzzerPackage)
	}
	fuzzerDir := filepath.Dir(walker.fuzzerPath)
	filesInFuzzerDir, err := os.ReadDir(fuzzerDir)
    if err != nil {
        panic(err)
    }
 
    for _, file := range filesInFuzzerDir {
    	fi, err := os.Stat(filepath.Join(fuzzerDir, file.Name()))
    	if err != nil {
    		panic(err)
    	}
    	if !fi.Mode().IsRegular() {
    		continue
    	}
    	walker.RewriteFile(filepath.Join(fuzzerDir, file.Name()), fuzzerFuncName)
    }
	walker.overlayArgs = walker.CreateOverlayFile(flagOverlay)
}
