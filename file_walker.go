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
)

var (
	customTestingName = "customFuzzTestingPkg"

	buildFlags2 = []string{
		"-buildmode", "c-archive",
		"-trimpath",
		"-gcflags", "all=-d=libfuzzer",
	}
	stdLibPkgs = []string{
		"slices",
		"slices",
		"cmp.test",
		"cmp",
		"os",
		"sync",
		"unsafe",
		"internal/race",
		"unsafe",
		"runtime",
		"internal/chacha8rand",
		"internal/goarch",
		"unsafe",
		"internal/chacha8rand_test",
		"bytes",
		"errors",
		"unsafe",
		"internal/reflectlite",
		"internal/abi",
		"internal/goarch",
		"unsafe",
		"internal/abi_test",
		"strings",
		"errors",
		"errors_test",
		"testing",
		"flag",
"context",
"time",
"runtime",
"runtime/internal/math",
"runtime/internal/math_test",
"testing/internal/testdeps",
"os/signal",
"sync",
"internal/race",
"runtime_test",
"internal/goos",
"math/rand",
"internal/godebug",
"sync/atomic",
"sync/atomic_test",
"reflect",
"strconv",
"math",
"internal/cpu",
"internal/cpu_test",
"internal/cpu.test",
"reflect_test",
"testing/quick",
"reflect.test",
"reflect_test",
"unicode/utf8",
"unicode/utf8_test",
"unicode",
"unicode_test",
"sort",
"math/bits",
"math/bits_test",
"fmt",
"fmt_test",
"unicode/utf8",
"unicode/utf8.test",
"unicode/utf8_test",
"testing/iotest",
"encoding/binary",
"io",
"reflect",
"encoding/binary_test",
"encoding/binary",
"encoding/binary.test",
"encoding/base64_test",
"encoding/base64",
"encoding/base64.test",
"math",
"strconv",
"encoding",
"sort",
"unicode/utf16",
"unicode/utf16_test",
"unicode",
"internal/testenv",
"internal/testenv_test",
"path/filepath",
"sort",
"io/fs",
"io/fs",
"path",
"internal/bytealg",
"internal/cpu",
"net",
"vendor/golang.org/x/net/dns/dnsmessage",
"internal/poll",
"internal/syscall/unix",
"syscall",
"internal/itoa_test",
"internal/itoa",
"internal/itoa.test",
"internal/oserror",
"syscall_test",
"os/exec",
"syscall",
"syscall.test",
"syscall_test",
"internal/syscall/execenv",
"internal/syscall/unix",
"os/exec_test",
"os/user",
"runtime/cgo",
"runtime/internal/sys",
"runtime/internal/sys",
"runtime/internal/sys_test",
"crypto/ecdsa",
"crypto/aes",
"crypto/internal/alias",
"crypto/internal/alias",
"crypto/internal/alias.test",
"crypto/internal/alias",
"crypto/subtle",
"crypto/subtle",
"crypto/subtle_test",
"crypto/rand",
"crypto/rand",
"crypto/internal/boring",
"crypto/internal/boring",
"crypto",
"crypto",
"hash",
"hash",
"hash_test",
"crypto/sha256",
"crypto/sha256",
"hash",
"hash.test",
"hash_test",
"hash",
"crypto",
"crypto_test",
"crypto/aes",
"crypto/aes.test",
"crypto/aes",
"crypto/cipher",
"crypto/cipher",
"crypto/internal/alias",
"crypto/subtle",
"crypto/subtle.test",
"crypto/subtle",
"crypto/subtle_test",
"crypto/cipher_test",
"crypto/aes",
"crypto/des",
"crypto/des",
"crypto/cipher",
"crypto/cipher.test",
"encoding/json",
"encoding/json",
"encoding/json_test",
"encoding/json",
"encoding/json.test",
"encoding/json",
"encoding/json_test",
"log",
"log",
"log/internal",
"log/internal",
"log_test",
"log",
"log.test",
"log_test",
"log",
"net/url",
"net/url",
"net/url_test",
"log",
"net/url",
"net/url.test",
"net/ur",
"mime",
"mime",
"bufio",
"mime_test",
"mime",
"mime.test",
"mime",
"mime_test",
"container/list",
"container/list",
"container/list_test",
"container/list",
"container/list.test",
"container/list_test",
"container/list",
"log",
"net/textproto",
"net/textproto",
"bufio",
"net/textproto.test",
"net/textproto",
"net/http",
"net/http",
"net/http/internal",
"net/http/internal",
"bufio",
"bufio.test",
"bufio                                                                                                                                                        ",
"net/http/internal.test                                                                                                                                             ",
"net/http/internal",
"regexp",
"regexp",
"regexp/syntax",
"regexp/syntax",
"regexp/syntax.test",
"regexp/syntax",
"regexp_test",
"regexp",
"regexp.test",
"regexp",
"regexp_test",
"net/http/internal",
"testing/fstest",
"testing/fstest",
"testing/fstest.test",
"testing/fstest",
"net/http/cookiejar",
"net/http/cookiejar",
"net/http",
"net/http.test",
"net/http",
"net/http_test",
"net/http/internal/ascii",
"net/url",
"net/http/cookiejar_test",
"log",
"net/http",
"net/http/cookiejar",
"net/http/cookiejar.test",
"net/http/cookiejar",
"net/http/cookiejar_test",
"net/http/httptest",
"net/http/httptest",
"compress/gzip",
"compress/gzip",
"hash/crc32",
"hash/crc32",
"hash/crc32_test",
"hash/crc32",
"hash/crc32.test",
"hash/crc32",
"hash/crc32_test",
"bufio",
"compress/flate",
"compress/flate",
"bufio",
"compress/flate_test",
"log",
"compress/flate",
"compress/flate.test",
"compress/flate",
"compress/flate_test",
"compress/gzip_test",
"net/http",
"net/http_test",
"regexp",
"regexp",
"regexp/syntax",
"regexp/syntax",
"regexp/syntax.test",
"regexp/syntax",
"regexp_test",
"regexp",
"regexp.test",
"regexp",
"crypto/internal/bigmod.test",
"crypto/internal/bigmod",
"crypto/internal/randutil",
"crypto/internal/randutil",
"math/big",
"crypto/internal/boring/bbig",
"crypto/internal/boring/bbig",
"math/big",
"crypto/rsa_test",
"crypto/x509",
"crypto/x509_test",
"crypto/elliptic",
"crypto/elliptic",
"math/big",
"crypto/internal/nistec",
"crypto/internal/nistec",
"crypto/internal/nistec/fiat",
"crypto/internal/nistec/fiat",
"crypto/internal/nistec/fiat_test",
"crypto/internal/nistec/fiat",
"crypto/internal/nistec/fiat.test",
"crypto/internal/nistec/fiat_test",
"crypto/internal/nistec/fiat",
"embed",
"embed",
"embed_test",
"net/http",
"embed",
"embed.test",
"embed_test",
"embed",
"log",
"crypto/internal/nistec_test",
"math/big",
"crypto/elliptic",
"crypto/elliptic.test",
"crypto/elliptic",
"crypto/internal/nistec",
"crypto/internal/nistec.test",
"crypto/internal/nistec",
"crypto/internal/nistec_test",
"crypto/ed25519",
"maps.test",
"maps",
"maps_test",
"net/netip",
"net/netip",
"internal/intern",
"internal/intern",
"internal/intern.test",
"internal/intern",
"net/netip_test",
"internal/intern",
"net/netip",
"net/netip.test",
"net/netip",
"net/netip_test",
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
		newName := strings.TrimSuffix(renamedTestFile, "_libFuzzer.go") + "_test.go"
		err := os.Rename(renamedTestFile, newName)
		if err != nil {
			panic(err)
		}
	}
	for originalFilePath, tmpFilePath := range walker.originalFiles {
		fmt.Println("Renaming ", originalFilePath, tmpFilePath, "...")
		err := os.Rename(tmpFilePath, originalFilePath)
		if err != nil {
			panic(err)
		}
	}
	os.RemoveAll(walker.tmpDir)
}

// "path" is expected to be a file in a module
// that a fuzzer uses.
func (walker *FileWalker) RewriteFile(path string) {
	originalFileContents, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	rewroteTestingFParams := walker.rewriteTestingFFunctionParams(path)
	if rewroteTestingFParams {
		err := walker.addShimImport(path)
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
	}
	// rename test files from *_test.go to *_libFuzzer.go
	if path[len(path)-8:] == "_test.go" {
		newName := strings.TrimSuffix(path, "_test.go") + "_libFuzzer.go"
		err := os.Rename(path, newName)
		if err != nil {
			panic(err)
		}
		// Store the new name
		if !stringInSlice(newName, walker.renamedTestFiles) {
			walker.renamedTestFiles = append(walker.renamedTestFiles, newName)
		}
	}
}

// Rewrites testing import of a single path
func (walker *FileWalker) addShimImport(path string) error {
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

	// Replace import path
	astutil.DeleteImport(fset, fCheck, "testing")
	astutil.AddNamedImport(fset,
		fCheck,
		"_",
		"testing")
	astutil.AddNamedImport(fset,
		fCheck,
		customTestingName,
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

// Checks whether a fuzz test exists in a given file
func (walker *FileWalker) rewriteTestingFFunctionParams(path string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}
	updated := false
	for _, decl := range f.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			for _, param := range funcDecl.Type.Params.List {
				if paramType, ok := param.Type.(*ast.StarExpr); ok {
					if p2, ok := paramType.X.(*ast.SelectorExpr); ok {
						if p3, ok := p2.X.(*ast.Ident); ok {
							if p3.Name == "testing" && p2.Sel.Name == "F" {
								p3.Name = customTestingName
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
}

func (walker *FileWalker) RewriteAllImportedTestFiles(files []string) error {
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
}

func (walker *FileWalker) RestoreRenamedTestFiles() error {
	for originalFile, renamedFile := range walker.renamedFiles {
		err := os.Rename(renamedFile, originalFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func (walker *FileWalker) addRenamedFile(oldPath, newPath string) {
	if _, ok := walker.renamedFiles[oldPath]; ok {
		panic("The file already exists which it shouldn't")
	}
	walker.renamedFiles[oldPath] = newPath
}

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
func GetAllSourceFilesOfFile(pkgPath, filePath string) ([]string, error) {
	files := make([]string, 0)
	pkgs, err := getAllPackagesOfFile(pkgPath, filePath)
	if err != nil {
		return files, err
	}
	for _, pkg := range pkgs {
		fmt.Println("PPPPPPPPPKKKKKKKKKKKKKGGGGGGGGGGGG: ", pkg.Name)
		for _, file := range pkg.GoFiles {
			fmt.Println("file: ", file)
			// There may be compiled files in the go cache. Ignore those
			if strings.Contains(file, "/.cache/") {
				continue
			}
			files = append(files, file)
		}
	}
	return files, nil
}

func getAllPackagesOfFile(pkgPath, filePath string) ([]*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags2,
		Tests:      true,
	}, "file="+filePath)
	if err != nil {
		return pkgs, err
	}
	err = os.Chdir(filepath.Dir(filePath))
	if err != nil {
		return pkgs, err
	}
	// There should only be one file
	if len(pkgs) != 1 {
		panic("there should only be one file here")
	}
	fmt.Println("appending pkg imports")
	return appendPkgImports(pkgs[0], pkgs, pkgPath)
}

func isStdLibPkg(importName string) bool {
	for _, stdLibPkg := range stdLibPkgs {
		if strings.EqualFold(importName, stdLibPkg) {
			return true
		}
	}
	return false
}

func appendPkgImports(pkg *packages.Package, pkgs []*packages.Package, modulePath string) ([]*packages.Package, error) {
	pkgsCopy := pkgs
	for _, imp := range pkg.Imports {
		// Check that the package is the same module
		if imp.Module != nil {
			if len(imp.Module.Path) < len(modulePath) {
				fmt.Println("skipping1 ", imp.Module.Path)
				continue
			}
			if imp.Module.Path != modulePath {
				fmt.Println("skipping2 ", imp.Module.Path)
				continue
			}
		}
		if isStdLibPkg(imp.PkgPath) {
			continue
		}

		fmt.Println(imp.PkgPath)
		p, err := loadPkg(imp.PkgPath)
		if err != nil {
			return pkgsCopy, err
		}
		for _, pack := range p {
			if pkgInPkgs(pack.PkgPath, pkgsCopy) {
				continue
			}
			fmt.Println(pack.PkgPath)
			pkgsCopy = append(pkgsCopy, pack)
			pkgsCopy, err = appendPkgImports(pack, pkgsCopy, modulePath)
			if err != nil {
				return pkgsCopy, err
			}
		}
	}
	return pkgsCopy, nil
}

func loadPkg(path string) ([]*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode:       LoadMode,
		BuildFlags: buildFlags2,
		Tests:      true,
	}, path)
	if err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

func pkgInPkgs(importPath string, pkgs []*packages.Package) bool {
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
