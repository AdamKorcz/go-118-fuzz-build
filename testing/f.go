package testing

import (
	"fmt"
	"os"
	"reflect"
	"github.com/AdamKorcz/go-118-fuzz-build/input"
)

type F struct {
	s *input.Source
	TempDirs []string
}

func NewF(data []byte) *F {
	return &F{s: input.NewSource(data), TempDirs: make([]string, 0)}
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
