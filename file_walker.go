package main

import (
	"bytes"
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

type FileWalker struct {
	renamedFiles     map[string]string
	renamedTestFiles []string
	rewrittenFiles   []string
	// Stores the original files
	originalFiles map[string]string
	tmpDir        string
}

func NewFileWalker() *FileWalker {
	tmpDir, err := os.MkdirTemp("", "gofuzzbuild")
	if err != nil {
		panic(err)
	}
	return &FileWalker{
		renamedFiles:     make(map[string]string),
		renamedTestFiles: make([]string, 0),
		rewrittenFiles:   make([]string, 0),
		originalFiles:    make(map[string]string),
		tmpDir:           tmpDir,
	}
}

func (walker *FileWalker) cleanUp() {
	for _, renamedTestFile := range walker.renamedTestFiles {
		fmt.Println("Cleaning up1... ", renamedTestFile)
		newName := strings.TrimSuffix(renamedTestFile, "_libFuzzer.go") + "_test.go"
		err := os.Rename(renamedTestFile, newName)
		if err != nil {
			panic(err)
		}
	}
	for originalFilePath, tmpFilePath := range walker.originalFiles {
		fmt.Println("Cleaning up2... ", originalFilePath, tmpFilePath, "...")
		err := os.Rename(tmpFilePath, originalFilePath)
		if err != nil {
			panic(err)
		}
		fmt.Println("renamed ", tmpFilePath, "to ", originalFilePath)
	}
	fmt.Println("removing walker.tmpDir..")
	err := os.RemoveAll(walker.tmpDir)
	if err != nil {
		panic(err)
	}
	fmt.Println("Removed walker.tmpDir")
}

// "path" is expected to be a file in a module
// that a fuzzer uses.
func (walker *FileWalker) RewriteFile(path, fuzzerPath string) {
	// Check for files outside of the fuzzing module.
	// This is quite late to catch it and should be done smarter and
	// earlier in the process.
	// This only catches an issue in the OSS-Fuzz env.
	// We should essentially check if the file is outside of the module dir.
	rewroteFile := false
	if path[len(path)-8:] == "_test.go" {
		if filepath.Dir(path) != filepath.Dir(fuzzerPath) {
			return
		}
	}
	if strings.Contains(path, "/root/.go/") {
		return
	}

	// TODO: Check if it is a "_test" pkg outside of the fuzzers dir.
	// If it is, then we should not rewrite it.
	fset1 := token.NewFileSet()
	//f, err := parser.ParseFile(fset1, path, nil, parser.PackageClauseOnly)
	parsedFile, err := parser.ParseFile(fset1, path, nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Check ends in "_test".
	// Could use "HasSuffix here instead"
	if len(parsedFile.Name.Name) >= 5 && parsedFile.Name.Name[len(parsedFile.Name.Name)-5:] == "_test" {
		if filepath.Dir(path) != filepath.Dir(fuzzerPath) {
			return
		}
	}

	// If it is a non-_test.go file that imports "testing",
	// we rewrite the testing param, since there is a high
	// chance that this is a utility package for fuzzing
	/*fsetCheck := token.NewFileSet()
	fCheck, err := parser.ParseFile(fsetCheck, path, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}*/
	for _, imp := range parsedFile.Imports {
		fmt.Println("import : ", imp.Path.Value)
		if imp.Path.Value == "\"testing\"" {
			astutil.DeleteImport(fset1, parsedFile, "testing")
			astutil.AddImport(fset1,
				parsedFile,
				"github.com/AdamKorcz/go-118-fuzz-build/testing")
			rewroteFile = true
		}
	}

	originalFileContents, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	//rewroteTestingFParams := walker.rewriteTestingFFunctionParams(path)
	// Here we should check if the file uses non-testing.F-related functions
	// If it does, then the file should have both our shim import
	// and the std lib "testing" import. If it doesn't then the file
	// should only have our custom testing import
	//usesNonFuzzTestingFuncs := something()
	//testingTWalker := NewTestingTWalker(path)
	/*fset := token.NewFileSet()
	fCheck, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}*/
	//ast.Walk(testingTWalker, parsedFile)

	if rewroteFile {
		f, err := os.CreateTemp(walker.tmpDir, "")
		if err != nil {
			panic(err)
		}
		_, err = f.Write(originalFileContents)
		if err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
		walker.originalFiles[path] = f.Name()

		var buf bytes.Buffer
		printer.Fprint(&buf, fset1, parsedFile)

		f2, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			panic(err)
		}
		defer f2.Close()
		f2.WriteString(string(buf.Bytes()))

		if !stringInSlice(path, walker.rewrittenFiles) {
			walker.rewrittenFiles = append(walker.rewrittenFiles, path)
		}
		fmt.Println("Overwrote ", path)
	}

	if path[len(path)-8:] == "_test.go" {
		if filepath.Dir(path) != filepath.Dir(fuzzerPath) {
			return
		}
		newName := strings.TrimSuffix(path, "_test.go") + "_libFuzzer.go"
		err := os.Rename(path, newName)
		if err != nil {
			panic(err)
		}
		// Store the new name
		if !stringInSlice(newName, walker.renamedTestFiles) {
			walker.renamedTestFiles = append(walker.renamedTestFiles, newName)
		}
		if strings.EqualFold(path, fuzzerPath) {
			originalFileContents2, err := os.ReadFile(newName)
			if err != nil {
				panic(err)
			}
			fmt.Println("originalFileContents2", string(originalFileContents2))
		}
	}
	/*if rewroteTestingFParams {
		//print("rewrote testingFparams in ", path)
		err := walker.addShimImport(path, testingTWalker.hasTestingT)
		if err != nil {
			panic(err)
		}
		// Save original file contents
		f, err := os.CreateTemp(walker.tmpDir, "")
		if err != nil {
			panic(err)
		}
		_, err = f.Write(originalFileContents)
		if err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
		walker.originalFiles[path] = f.Name()
	}*/
	// rename test files from *_test.go to *_libFuzzer.go
	/*if path[len(path)-8:] == "_test.go" {
		if filepath.Dir(path) != filepath.Dir(fuzzerPath) {
			return
		}
		newName := strings.TrimSuffix(path, "_test.go") + "_libFuzzer.go"
		err := os.Rename(path, newName)
		if err != nil {
			panic(err)
		}
		// Store the new name
		if !stringInSlice(newName, walker.renamedTestFiles) {
			walker.renamedTestFiles = append(walker.renamedTestFiles, newName)
		}
	}*/
}

// Rewrites testing import of a single path
func (walker *FileWalker) addShimImport(path string, hasTestingT bool) error {
	//fmt.Println("Rewriting ", path)
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

	/*if !hasTestingT {
		astutil.DeleteImport(fset, fCheck, "testing")
	}*/

	// Replace import path
	//astutil.DeleteImport(fset, fCheck, "testing")
	/*astutil.AddNamedImport(fset,
	fCheck,
	"_",
	"testing")*/
	/*astutil.AddNamedImport(fset,
	fCheck,
	//customTestingName,
	"testing",
	"github.com/AdamKorcz/go-118-fuzz-build/testing")*/
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

/*func NewTestingTWalker(filepath string) *TestingTWalker {
	return &TestingTWalker{
		file: filepath,
	}
}

type TestingTWalker struct {
	file        string
	hasTestingT bool
}

func (walker *TestingTWalker) Visit(n ast.Node) ast.Visitor {
	//fmt.Println("Visit'ing node")
	if n == nil {
		return walker
	}
	if p2, ok := n.(*ast.SelectorExpr); ok {
		if p3, ok := p2.X.(*ast.Ident); ok {
			if p3.Name == "testing" && p2.Sel.Name == "T" {
				walker.hasTestingT = true
			}
		}
	}
	return walker
}*/

// Checks whether a fuzz test exists in a given file
/*func (walker *FileWalker) rewriteTestingFFunctionParams(parsedFile *token.FileSet) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}
	updated := false
	for _, decl := range parsedFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			for _, param := range funcDecl.Type.Params.List {
				if paramType, ok := param.Type.(*ast.StarExpr); ok {
					if p2, ok := paramType.X.(*ast.SelectorExpr); ok {
						if p3, ok := p2.X.(*ast.Ident); ok {
							if p3.Name == "testing" && p2.Sel.Name == "F" {
								//p3.Name = customTestingName
								updated = true
							}
						}
					}
				}
			}
		}
	}
	if updated {
		var buf bytes.Buffer
		printer.Fprint(&buf, fset, f)

		newFile, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		defer newFile.Close()
		newFile.Write(buf.Bytes())

		if !stringInSlice(path, walker.rewrittenFiles) {
			walker.rewrittenFiles = append(walker.rewrittenFiles, path)
		}
	}
	return updated
}*/

/*func (walker *FileWalker) RewriteAllImportedTestFiles(files []string) error {
	for _, file := range files {
		if file[len(file)-8:] == "_test.go" {
			newName := strings.TrimSuffix(file, "_test.go") + "_libFuzzer.go"
			err := os.Rename(file, newName)
			if err != nil {
				return err
			}
			walker.addRenamedFile(file, newName)
		}
	}
	return nil
}*/

func (walker *FileWalker) RestoreRenamedTestFiles() error {
	for originalFile, renamedFile := range walker.renamedFiles {
		err := os.Rename(renamedFile, originalFile)
		if err != nil {
			return err
		}
	}
	return nil
}

/*func (walker *FileWalker) addRenamedFile(oldPath, newPath string) {
	if _, ok := walker.renamedFiles[oldPath]; ok {
		panic("The file already exists which it shouldn't")
	}
	walker.renamedFiles[oldPath] = newPath
}*/

// Gets the path of
func getPathOfFuzzFile(pkgPath, fuzzerName string, buildFlags []string) (string, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags,
		Tests:      true,
	}, "pattern="+pkgPath)
	if err != nil {
		return "", err
	}
	for _, pkg := range pkgs {
		if pkg.PkgPath != pkgPath {
			continue
		}
		for _, file := range pkg.GoFiles {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file, nil, 0)
			if err != nil {
				return "", err
			}
			for _, decl := range f.Decls {
				if _, ok := decl.(*ast.FuncDecl); ok {
					if decl.(*ast.FuncDecl).Name.Name == fuzzerName {
						return file, nil

					}
				}
			}
		}
	}
	return "", fmt.Errorf("Could not find the fuzz func")
}

/* Gets a list of files that are imported by a file */
func GetAllSourceFilesOfFile(modulePath, fuzzerFilePath string) ([]string, error) {
	files := make([]string, 0)
	pkgs, err := getAllPackagesOfFile(modulePath, fuzzerFilePath)
	if err != nil {
		return files, err
	}
	for _, pkg := range pkgs {
		//fmt.Println("PPPPPPPPPKKKKKKKKKKKKKGGGGGGGGGGGG: ", pkg.Name)
		for _, file := range pkg.GoFiles {
			if strings.Contains(file, "pkg/fuzz") {
				fmt.Println("file in this pkg: ", file)
			}
			// There may be files in the go cache. Ignore those
			if strings.Contains(file, "/.cache/go-build") {
				continue
			}
			files = append(files, file)
		}
	}
	return files, nil
}

func getAllPackagesOfFile(modulePath, fuzzerFilePath string) ([]*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags2,
		Tests:      true,
	}, "file="+fuzzerFilePath)
	if err != nil {
		return pkgs, err
	}
	err = os.Chdir(filepath.Dir(fuzzerFilePath))
	if err != nil {
		return pkgs, err
	}
	// There should only be one file
	if len(pkgs) != 1 {
		panic("there should only be one file here")
	}
	//fmt.Println("appending pkg imports")
	fuzzerPkg := pkgs[0]
	return appendPkgImports(pkgs[0], fuzzerPkg, pkgs, modulePath, fuzzerFilePath)
}

func isStdLibPkg(importName string) bool {
	for _, stdLibPkg := range utils.StdLibPkgs {
		if strings.EqualFold(importName, stdLibPkg) {
			return true
		}
	}
	if len(importName) >= 6 && importName[:6] == "crypto" {
		return true
	}
	if len(importName) >= 7 && importName[:7] == "archive" {
		return true
	}
	if len(importName) >= 8 && importName[:8] == "internal" {
		return true
	}
	if len(importName) >= 2 && importName[:2] == "go" {
		return true
	}
	if len(importName) >= 8 && importName[:8] == "encoding" {
		return true
	}
	if len(importName) >= 8 && importName[:8] == "compress" {
		return true
	}
	if len(importName) >= 3 && importName[:3] == "net" {
		return true
	}
	if len(importName) >= 7 && importName[:7] == "testing" {
		return true
	}
	if len(importName) >= 8 && importName[:8] == "internal" {
		return true
	}
	if len(importName) >= 7 && importName[:7] == "runtime" {
		return true
	}
	return false
}

// We need this to get the .go files of all the imports
// so we can check if we need to rewrite any of the
// imported .go files.
// This is currently very slow to a degree that it could
// be a problem.
func appendPkgImports(pkg, fuzzerPkg *packages.Package, pkgs []*packages.Package, modulePath, fuzzerPath string) ([]*packages.Package, error) {
	pkgsCopy := pkgs
	for _, imp := range pkg.Imports {
		if strings.Contains(imp.PkgPath, "pkg/fuzz") {
			fmt.Println("WE GOT THE FUZZING UTILS")
		}
		// We might have already loaded this import package
		if alreadyHaveThisPkg(imp.PkgPath, pkgsCopy) {
			continue
		}
		// Check that the package is the same module
		// This is a performance optimization, so we
		// can skip it if we don't have the modules
		if imp.Module != nil && modulePath != "" {
			if len(imp.Module.Path) < len(modulePath) {
				continue
			}
			if imp.Module.Path != modulePath {
				continue
			}
		}
		if isStdLibPkg(imp.PkgPath) {
			continue
		}

		//fmt.Println("loading pkg: ", imp.PkgPath)
		p, err := loadPkg(imp.PkgPath)
		if err != nil {
			//fmt.Println("error loadPkg: ", err)
			return pkgsCopy, err
		}
		//fmt.Println("Len of loaded packages: ", len(pkgsCopy))
		for _, pack := range p {
			// Here we should evaluate if the package:
			// 1. is a "_test" package
			// 2. is imported (ie. it is not the package that the fuzzer is in)
			// 3. there are other packages in the folder for example a non-_test package
			// If the answer is "yes" to all three questions, then we should continue here
			if !shouldChangeTestPackage(imp, fuzzerPkg, fuzzerPath) {
				//fmt.Println("Should not rewrite, ", imp)
				continue
			}

			//fmt.Println("THIS PKG: ", pack.PkgPath, "FuzzerPath: ", fuzzerPath)
			if strings.Contains(imp.PkgPath, "pkg/fuzz") {
				fmt.Println("APPENDING THE FUZZING UTILS")
			}
			pkgsCopy = append(pkgsCopy, pack)
			pkgsCopy, err = appendPkgImports(pack, fuzzerPkg, pkgsCopy, modulePath, fuzzerPath)
			if err != nil {
				return pkgsCopy, err
			}
		}
	}
	return pkgsCopy, nil
}

func shouldChangeTestPackage(imp, fuzzerPkg *packages.Package, fuzzerPath string) bool {
	//fmt.Println("imp.Name:::::::::::::", imp.Name)
	if strings.HasSuffix(imp.Name, "_test") {
		//fmt.Println("return here: ", imp.Name)
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
	// If the import dir path is not the same as the fuzzers, then we shouldn't rewrite it
	if filepath.Dir(imp.GoFiles[0]) != filepath.Dir(fuzzerPath) {
		//fmt.Println("returning here. filpath.Dir(Imp.Gofiles[0]) = ", filepath.Dir(imp.GoFiles[0]), "filepath.Dir(fuzzerPath) = ", filepath.Dir(fuzzerPath))
		//return false
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

// rewriteTestingImports rewrites imports for:
// - all package files
// - the fuzzer
// - dependencies
//
// it rewrites "testing" => "github.com/AdamKorcz/go-118-fuzz-build/testing"
func rewriteTestingImports(pkgs []*packages.Package, fuzzName string) (string, []byte, error) {
	return "", []byte(""), nil
	/*var fuzzFilepath string
	var originalFuzzContents []byte
	originalFuzzContents = []byte("NONE")

	// First find file with fuzz harness
	for _, pkg := range pkgs {
		for _, file := range pkg.GoFiles {
			err := rewriteTestingImport(file)
			if err != nil {
				panic(err)
			}
		}
	}

	// rewrite testing in imported packages
	packages.Visit(pkgs, rewriteImportTesting, nil)

	for _, pkg := range pkgs {
		for _, file := range pkg.GoFiles {
			fuzzFile, b, err := rewriteFuzzer(file, fuzzName)
			if err != nil {
				panic(err)
			}
			if fuzzFile != "" {
				fuzzFilepath = fuzzFile
				originalFuzzContents = b
			}
		}
	}
	return fuzzFilepath, originalFuzzContents, nil*/
}

/*func rewriteFuzzer(path, fuzzerName string) (originalPath string, originalFile []byte, err error) {
	var fileHasOurHarness bool // to determine whether we should rewrite filename
	fileHasOurHarness = false

	var originalFuzzContents []byte
	originalFuzzContents = []byte("NONE")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return "", originalFuzzContents, err
	}
	for _, decl := range f.Decls {
		if _, ok := decl.(*ast.FuncDecl); ok {
			if decl.(*ast.FuncDecl).Name.Name == fuzzerName {
				fileHasOurHarness = true
			}
		}
	}

	if fileHasOurHarness {
		originalFuzzContents, err = os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		// Replace import path
		astutil.DeleteImport(fset, f, "testing")
		astutil.AddImport(fset, f, "github.com/AdamKorcz/go-118-fuzz-build/testing")
	}

	// Rewrite filename
	if fileHasOurHarness {
		var buf bytes.Buffer
		printer.Fprint(&buf, fset, f)

		newFile, err := os.Create(path + "_fuzz.go")
		if err != nil {
			panic(err)
		}
		defer newFile.Close()
		newFile.Write(buf.Bytes())
		return path, originalFuzzContents, nil
	}
	return "", originalFuzzContents, nil
}*/

// Rewrites testing import of a single path
/*func rewriteTestingImport(path string) error {
	//fmt.Println("Rewriting ", path)
	fsetCheck := token.NewFileSet()
	fCheck, err := parser.ParseFile(fsetCheck, path, nil, parser.ImportsOnly)
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

	// Replace import path
	for _, imp := range fCheck.Imports {
		if imp.Path.Value == "testing" {
			imp.Path.Value = "github.com/AdamKorcz/go-118-fuzz-build/testing"
		}
	}
	return nil
}*/

// Rewrites testing import of a package
/*func rewriteImportTesting(pkg *packages.Package) bool {
	for _, file := range pkg.GoFiles {
		err := rewriteTestingImport(file)
		if err != nil {
			panic(err)
		}
	}
	return true
}*/
