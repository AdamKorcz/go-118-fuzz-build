package testingtypes

import (
	"fmt"
	"strings"
	"time"
)

type T struct{}

func unsupportedApi(name string) string {
	plsOpenIss := "Please open an issue https://github.com/AdamKorcz/go-118-fuzz-build if you need this feature."
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s is not supported when fuzzing in libFuzzer mode\n.", name))
	b.WriteString(plsOpenIss)
	return b.String()
}

func (c *T) Cleanup(f func()) {
	f()
}
func (t *T) Deadline() (deadline time.Time, ok bool) {
	panic(unsupportedApi("t.Deadline()"))
}

func (c *T) Error(args ...any) {
	fmt.Println(args...)
	panic(unsupportedApi("Encoutered"))
}

func (c *T) Errorf(format string, args ...any) {}
func (c *T) Fail() {
	panic(unsupportedApi("t.Fail()"))
}

func (c *T) FailNow() {
	panic(unsupportedApi("t.FailNow()"))
}

func (c *T) Failed() bool {
	panic(unsupportedApi("t.Failed()"))
}

func (c *T) Fatal(args ...any) {
	fmt.Println(args...)
	panic("fatal")
}
func (c *T) Fatalf(format string, args ...any) {
	fmt.Println(format, args)
	panic("fatal")
}
func (c *T) Helper() {
	panic(unsupportedApi("t.Failed()"))
}
func (c *T) Log(args ...any) {
	fmt.Println(args...)
}

func (c *T) Logf(format string, args ...any) {
	fmt.Println(format)
	fmt.Println(args...)
}

func (c *T) Name() string {
	return "fuzzer"
}

func (t *T) Parallel() {
	panic(unsupportedApi("t.Failed()"))
}
func (t *T) Run(name string, f func(t *T)) bool {
	panic(unsupportedApi("t.Run()."))
}

func (t *T) Setenv(key, value string) {

}

func (c *T) Skip(args ...any) {
	panic(unsupportedApi("t.Skip()"))
}
func (c *T) SkipNow() {
	panic(unsupportedApi("t.SkipNow()"))
}
func (c *T) Skipf(format string, args ...any) {
	panic(unsupportedApi("t.Skipf()"))
}
func (c *T) Skipped() bool {
	panic(unsupportedApi("t.Skipped()"))
}
func (c *T) TempDir() string {
	panic(unsupportedApi("t.TempDir()"))
}
