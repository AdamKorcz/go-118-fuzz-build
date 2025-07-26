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
	fuzzGoContents = `package testing

import (
	"fmt"
	"os"
	"reflect"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type F struct {
	s *Source
	TempDirs []string
}

func NewF(data []byte) *F {
	return &F{s: NewSource(data), TempDirs: make([]string, 0)}
}

func (f *F) CleanupTempDirs() {
	for _, tempDir := range f.TempDirs {
		os.RemoveAll(tempDir)
	}
}

func (f *F) Add(args ...any)                   {}
func (c *F) Cleanup(f func())                  {}
func (c *F) Error(args ...any)                 {}
func (c *F) Errorf(format string, args ...any) {}
func (f *F) Fail()                             {}
func (c *F) FailNow()                          {}
func (c *F) Failed() bool                      { return false }
func (c *F) Fatal(args ...any)                 {}
func (c *F) Fatalf(format string, args ...any) {}
func (f *F) Fuzz(ff any) {
	f.s.FillAndCall(ff, reflect.ValueOf(new(T)))
}
func (f *F) Helper() {}
func (c *F) Log(args ...any) {
	fmt.Print(args...)
}
func (c *F) Logf(format string, args ...any) {
	fmt.Println(fmt.Sprintf(format, args...))
}
func (c *F) Name() string             { return "libFuzzer" }
func (c *F) Setenv(key, value string) {}
func (c *F) Skip(args ...any) {
	panic("GO-FUZZ-BUILD-PANIC")
}
func (c *F) SkipNow() {
	panic("GO-FUZZ-BUILD-PANIC")
}
func (c *F) Skipf(format string, args ...any) {
	panic("GO-FUZZ-BUILD-PANIC")
}
func (f *F) Skipped() bool { return false }

func (f *F) TempDir() string {
	dir, err := os.MkdirTemp("", "fuzzdir-")
	if err != nil {
		panic(err)
	}
	f.TempDirs = append(f.TempDirs, dir)

	return dir
}

// Source takes a byteslice, and arguments can be pulled from it.
type Source struct {
	s         []byte
	i         int64 // current reading index
	exhausted bool
}

func NewSource(data []byte) *Source {
	return &Source{data, 0, false}
}

// IsExhausted returns true if we tried to read more data than this source
// could deliver.
func (s *Source) IsExhausted() bool {
	return s.exhausted
}

// Len returns the number of bytes of the unread portion of the data.
func (s *Source) Len() int {
	if s.i >= int64(len(s.s)) {
		return 0
	}
	return int(int64(len(s.s)) - s.i)
}

// Used returns the number of bytes already consumed.
func (s *Source) Used() int {
	return int(s.i)
}

// Read implements the io.Reader interface.
func (s *Source) Read(b []byte) (n int, err error) {
	if s.i >= int64(len(s.s)) {
		n, err = 0, io.EOF
	} else {
		n = copy(b, s.s[s.i:])
		s.i += int64(n)
	}
	if n < len(b) {
		s.exhausted = true
	}
	return n, err
}

// getBytes returns a slice of size bytes, as a direct reference if possible.
func (s *Source) getBytes(size int) []byte {
	if end := int(s.i) + size; end < len(s.s) { // Fast-path, no-copy deliver
		pos := s.i
		s.i += int64(size)
		return s.s[pos:end]
	}
	// Slow path
	buf := make([]byte, size)
	s.Read(buf)
	return buf
}

// readInt reads a signed integer from the source
func (s *Source) readInt(num reflect.Kind) int64 {
	switch num {
	case reflect.Int8:
		return int64(int8(s.getBytes(1)[0]))
	case reflect.Int16:
		return int64(int16(binary.BigEndian.Uint16(s.getBytes(2))))
	case reflect.Int32:
		return int64(int32(binary.BigEndian.Uint32(s.getBytes(4))))
	case reflect.Int64, reflect.Int:
		return int64(binary.BigEndian.Uint64(s.getBytes(8)))
	}
	panic(fmt.Sprintf("unsupported type: %v", num))
}

// readUint reads an unsigned integer from the source
func (s *Source) readUint(num reflect.Kind) uint64 {
	switch num {
	case reflect.Uint8:
		return uint64(uint8(s.getBytes(1)[0]))
	case reflect.Uint16:
		return uint64(binary.BigEndian.Uint16(s.getBytes(2)))
	case reflect.Uint32:
		return uint64(binary.BigEndian.Uint32(s.getBytes(4)))
	case reflect.Uint, reflect.Uint64:
		return binary.BigEndian.Uint64(s.getBytes(8))
	}
	panic(fmt.Sprintf("unsupported type: %v", num))
}

// FillAndCall fills the argument for the given ff (which is supposed to be a function),
// and then invokes the function.
// It returns 'true' if the function was invoked. A return-value of false means
// that the method was not invoked: probably because of insufficient input.
func (s *Source) FillAndCall(ff any, arg0 reflect.Value) (ok bool) {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	args := make([]reflect.Value, method.NumIn())
	args[0] = arg0
	var dynamic []int
	// Fill all fixed-size arguments first, then dynamic-sized fields.
	for i := 1; i < method.NumIn(); i++ {
		v := method.In(i)
		if v.Kind() <= reflect.Float64 { // fixed-size
			args[i] = s.fillArg(v, 0)
		} else { // dynamic or panic later
			dynamic = append(dynamic, i)
		}
	}
	// Second loop to fill dynamic-sized stuff
	// For filling the dynamic fields.
	// If we have only one field, it should get all the remaining input.
	// If we have N, then,
	// 1. Read N bytes [b1, b2, b3 .. bn] .
	// 2. Let the relative weights of b determine how much of the
	//    remaining input that field n gets
	weights := s.getBytes(len(dynamic))
	sum := 0
	for _, v := range weights {
		sum += int(v)
	}
	bytesLeft := s.Len()
	for i, argNum := range dynamic {
		if i == len(dynamic)-1 { // last element, it get's all that if left
			args[argNum] = s.fillArg(method.In(argNum), s.Len())
			break
		}
		var argSize = bytesLeft / len(dynamic)
		if sum > 0 {
			argSize = (bytesLeft * int(weights[i])) / sum
		}
		args[argNum] = s.fillArg(method.In(argNum), argSize)
	}
	fn.Call(args)
	return true
}

func (s *Source) fillArg(v reflect.Type, max int) reflect.Value {
	newElem := reflect.New(v).Elem()
	switch k := v.Kind(); k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		newElem.SetInt(s.readInt(k))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		newElem.SetUint(s.readUint(k))
	case reflect.Float32:
		newElem.Set(reflect.ValueOf(math.Float32frombits(uint32(s.readUint(reflect.Uint32)))))
	case reflect.Float64:
		newElem.Set(reflect.ValueOf(math.Float64frombits(s.readUint(reflect.Uint64))))
	case reflect.Bool:
		newElem.Set(reflect.ValueOf(s.readUint(reflect.Uint8)&0x1 != 0))
	case reflect.String:
		newElem.SetString(string(s.getBytes(max)))
	case reflect.Slice:
		if v.Elem().Kind() == reflect.Uint8 { // []byte
			newElem.SetBytes(s.getBytes(max))
		} else {
			panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
	}
	return newElem
}
`
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

	fuzzGoFile, err := os.CreateTemp(walker.tmpDir, "")
	if err != nil {
		panic(err)
	}
	if _, err := fuzzGoFile.Write([]byte(fuzzGoContents)); err != nil {
		fuzzGoFile.Close()
		panic(err)
	}
	fuzzGoFile.Close()
	newOverlayMap.Replace["/src/.go/src/testing.fuzz.go"] = fuzzGoFile.Name()


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
