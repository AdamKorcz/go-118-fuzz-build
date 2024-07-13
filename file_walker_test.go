package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRewriteFuncTestingFParams(t *testing.T) {
	fileContents := `package main
import (
	"testing"
)
func ourTestHelper(f *testing.F) {
	_ = f
}`
	expectedFileContents := `package main

import (
	"testing"
)

func ourTestHelper(f *customFuzzTestingPkg.F) {
	_ = f
}
`
	file := filepath.Join(t.TempDir(), "file.go")
	err := os.WriteFile(file, []byte(fileContents), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	rewriteFuzzerImports(file)
	gotFileContents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotFileContents, []byte(expectedFileContents)) {
		t.Errorf("%s", cmp.Diff(gotFileContents, []byte(expectedFileContents)))
	}

}
