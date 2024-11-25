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

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"github.com/AdamKorcz/go-118-fuzz-build/utils"
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
	if walker.sanitizer == "coverage" {
		os.Remove(strings.TrimSuffix(walker.fuzzerPath, "_test.go") + "_libFuzzer.go")
	}
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

func (walker *FileWalker) ignorePath(path string) bool {
	// Let's not rewrite dependencies in "/root/go/pkg/mod" for now.
	// They are a challenge in itself.
	if strings.HasPrefix(path, "/root/go/pkg/mod") {
		return true
	}
	if path[len(path)-8:] == "_test.go" {
		if filepath.Dir(path) != filepath.Dir(walker.fuzzerPath) {
			return true
		}
	}
	if strings.Contains(path, "/root/.go/") {
		return true
	}

	//TODO: CHECK IF THIS IS IN OUR go-118-fuzz-build module in a better way
	if strings.Contains(path, "go-118-fuzz-build/testing") {
		return true
	}
	return false
}

func (walker *FileWalker) createRewrittenHarness(path string, fset1 *token.FileSet, parsedFile *ast.File) error {
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
	fmt.Println("Creating new fuzzer on ", visibleFuzzerPath)
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
	return nil
}

// "path" is expected to be a file in a module
// that a fuzzer uses.
func (walker *FileWalker) RewriteFile(path, fuzzFuncName string) {
	// Check for files outside of the fuzzing module.
	// This is quite late to catch it and should be done smarter and
	// earlier in the process.
	// This only catches an issue in the OSS-Fuzz env.
	// We should essentially check if the file is outside of the module dir.

	if walker.ignorePath(path) {
		return
	}

	// TODO: Check if it is a "_test" pkg outside of the fuzzers dir.
	// If it is, then we should not rewrite it.
	fset1 := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset1, path, nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Check ends in "_test".
	// Could use "HasSuffix here instead"
	if len(parsedFile.Name.Name) >= 5 && parsedFile.Name.Name[len(parsedFile.Name.Name)-5:] == "_test" {
		fmt.Println("sourcefile: ", path)
		if filepath.Dir(path) != filepath.Dir(walker.fuzzerPath) {
			return
		}
	}

	// If it is a non-_test.go file that imports "testing",
	// we rewrite the testing param, since there is a high
	// chance that this is a utility package for fuzzing
	rewroteFile := false
	for _, imp := range parsedFile.Imports {
		if imp.Path.Value == "\"testing\"" {
			astutil.DeleteImport(fset1, parsedFile, "testing")
			astutil.AddImport(fset1,
				parsedFile,
				"github.com/AdamKorcz/go-118-fuzz-build/testing")
			rewroteFile = true
			fmt.Println("Rewrote ", path)
		}
	}

	// If coverage: prepend "F"
	if walker.sanitizer == "coverage" && strings.EqualFold(path, walker.fuzzerPath) {

		// Change fuzz function name from Fuzz* to FFuzz*
		for _, decl := range parsedFile.Decls {
			if _, ok := decl.(*ast.FuncDecl); ok {
				if decl.(*ast.FuncDecl).Name.Name == fuzzFuncName {
					fmt.Printf("changing func name from %s to %s", decl.(*ast.FuncDecl).Name.Name, fmt.Sprintf("F%s", fuzzFuncName))
					decl.(*ast.FuncDecl).Name.Name = fmt.Sprintf("F%s", fuzzFuncName)
				}
			}
		}

		// Make a copy of the original fuzzer contents
		err = walker.createRewrittenHarness(path, fset1, parsedFile)
		if err != nil {
			panic(err)
		}
	} else if rewroteFile {
		f, err := os.CreateTemp(walker.tmpDir, "")
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		printer.Fprint(&buf, fset1, parsedFile)

		_, err = f.Write(buf.Bytes())
		if err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
		var keyName string
		if strings.EqualFold(path, walker.fuzzerPath) {
			err = walker.createRewrittenHarness(path, fset1, parsedFile)
			if err != nil {
				panic(err)
			}
		} else if path[len(path)-8:] == "_test.go" && filepath.Dir(path) == filepath.Dir(walker.fuzzerPath) {
			keyName = strings.TrimSuffix(path, "_test.go") + "_libFuzzer.go"
			walker.overlayMap.Replace[keyName] = f.Name()
		} else {
			keyName = path
			walker.overlayMap.Replace[keyName] = f.Name()
		}
	}

	if path[len(path)-8:] == "_test.go" {
		// We should not substitute the fuzzer in an overlay map.
		// It creates problems in the coverage build.
		// Instead we should create the modified fuzzer in its place
		if path == walker.fuzzerPath {
			return
		}
		if filepath.Dir(path) != filepath.Dir(walker.fuzzerPath) {
			return
		}
		newName := strings.TrimSuffix(path, "_test.go") + "_libFuzzer.go"
		err := os.Rename(path, newName)
		if err != nil {
			panic(err)
		}
		// Store the new name
		walker.renamedTestFiles[path] = newName
	}
}

// Rewrites testing import of a single path
func (walker *FileWalker) addShimImport(path string, hasTestingT bool) error {
	fset := token.NewFileSet()
	fCheck, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}

	// First check if the import already exists
	// Return if it does.
	for _, imp := range fCheck.Imports {
		if imp.Path.Value == "github.com/AdamKorcz/go-118-fuzz-build/testing" {
			return nil
		}
	}
	astutil.DeleteImport(fset, fCheck, "testing")
	astutil.AddImport(fset,
		fCheck,
		//customTestingName,
		"github.com/AdamKorcz/go-118-fuzz-build/testing")
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, fCheck)

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(string(buf.Bytes()))

	if !stringInSlice(path, walker.rewrittenFiles) {
		walker.rewrittenFiles = append(walker.rewrittenFiles, path)
	}
	return nil
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

/* Gets a list of files that are imported by a file */
func (walker *FileWalker) GetAllSourceFilesOfFile(modulePath string) error {
	//files := make([]string, 0)
	pkgs, err := walker.getAllPackagesOfFile(modulePath)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.GoFiles {
			// There may be files in the go cache. Ignore those
			if strings.Contains(file, "/.cache/go-build") {
				continue
			}
			walker.allFiles = append(walker.allFiles, file)
		}
	}
	return nil
}

func (walker *FileWalker) getAllPackagesOfFile(modulePath string) ([]*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags2,
		Tests:      true,
	}, "file="+walker.fuzzerPath)

	if err != nil {
		return pkgs, err
	}
	err = os.Chdir(filepath.Dir(walker.fuzzerPath))
	if err != nil {
		return pkgs, err
	}
	// There should only be one file
	if len(pkgs) != 1 {
		fmt.Println(pkgs[0])
		panic("there should only be one file here")
	}
	fuzzerPkg := pkgs[0]
	return appendPkgImports(pkgs[0], fuzzerPkg, pkgs, modulePath)
}

// We need this to get the .go files of all the imports
// so we can check if we need to rewrite any of the
// imported .go files.
// This is currently very slow to a degree that it could
// be a problem.
func appendPkgImports(pkg, fuzzerPkg *packages.Package, pkgs []*packages.Package, modulePath string) ([]*packages.Package, error) {
	pkgsCopy := pkgs
	for _, imp := range pkg.Imports {
		// We might have already loaded this import package
		if alreadyHaveThisPkg(imp.PkgPath, pkgsCopy) {
			continue
		}
		// Check that the package is the same module
		// This is a performance optimization, so we
		// can skip it if we don't have the modules
		/*if imp.Module != nil && modulePath != "" {
			if len(imp.Module.Path) < len(modulePath) {
				continue
			}
			if imp.Module.Path != modulePath {
				continue
			}
		}*/
		if utils.IsStdLibPkg(imp.PkgPath) {
			continue
		}
		// Could we make some more static checks here to speed up things?
		p, err := loadPkg(imp.PkgPath)
		if err != nil {
			// We don't do anything in this case, since this
			// may happen for modules we don't have on the
			// system. In most cases, it doesn't matter, so
			// let's optimize when this is actually a pain
			// for someone.
			continue
			return pkgsCopy, err
		}
		for _, pack := range p {
			// Here we should evaluate if the package:
			// 1. is a "_test" package
			// 2. is imported (ie. it is not the package that the fuzzer is in)
			// 3. there are other packages in the folder for example a non-_test package
			// If the answer is "yes" to all three questions, then we should continue here
			if !shouldChangeTestPackage(imp, fuzzerPkg) {
				//fmt.Println("Should not rewrite, ", imp)
				continue
			}

			pkgsCopy = append(pkgsCopy, pack)
			pkgsCopy, err = appendPkgImports(pack, fuzzerPkg, pkgsCopy, modulePath)
			if err != nil {
				return pkgsCopy, err
			}
		}
	}
	return pkgsCopy, nil
}

func shouldChangeTestPackage(imp, fuzzerPkg *packages.Package) bool {
	if strings.HasSuffix(imp.Name, "_test") {
		return false
	}
	// Get the filepath of the package
	for i, _ := range imp.GoFiles {
		if i == 0 {
			continue
		}
		if filepath.Dir(imp.GoFiles[i]) != filepath.Dir(imp.GoFiles[i-1]) {
			panic("We have files outside of the package dir")
		}
	}

	return true
}

func loadPkg(path string) ([]*packages.Package, error) {
	loadMode := packages.NeedName |
		packages.NeedFiles |
		packages.NeedImports |
		packages.NeedModule
	pkgs, err := packages.Load(&packages.Config{
		Mode:       loadMode,
		BuildFlags: buildFlags2,
		Tests:      true,
	}, path)
	if err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

func alreadyHaveThisPkg(importPath string, pkgs []*packages.Package) bool {
	for _, pkg := range pkgs {
		if strings.EqualFold(pkg.PkgPath, importPath) {
			return true
		}
	}
	return false
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
	err := walker.GetAllSourceFilesOfFile(modulePath)
	if err != nil {
		panic(err)
	}
	for _, sourceFile := range walker.allFiles {
		walker.RewriteFile(sourceFile, fuzzerFuncName)
	}
	if walker.sanitizer == "coverage" {
		walker.createCoverageRunner(fuzzerFuncName, fuzzerPackage)
	}
	walker.overlayArgs = walker.CreateOverlayFile(flagOverlay)
}
