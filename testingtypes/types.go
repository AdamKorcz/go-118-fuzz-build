package testingtypes

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type T struct {
	CreatedDirs 	[]string
}

func unsupportedApi(name string) string {
	plsOpenIss := "Please open an issue https://github.com/AdamKorcz/go-118-fuzz-build if you need this feature."
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s is not supported when fuzzing in libFuzzer mode\n.", name))
	b.WriteString(plsOpenIss)
	return b.String()
}

func (t *T) Cleanup(f func()) {
	f()
}
func (t *T) Deadline() (deadline time.Time, ok bool) {
	panic(unsupportedApi("t.Deadline()"))
}

func (t *T) Error(args ...any) {
	fmt.Println(args...)
	panic("error")
}

func (t *T) Errorf(format string, args ...any) {
	fmt.Println(format)
	fmt.Println(args...)
	panic("errorf")
}

func (t *T) Fail() {
	panic(unsupportedApi("t.Fail()"))
}

func (t *T) FailNow() {
	panic(unsupportedApi("t.FailNow()"))
}

func (t *T) Failed() bool {
	panic(unsupportedApi("t.Failed()"))
}

func (t *T) Fatal(args ...any) {
	fmt.Println(args...)
	panic("fatal")
}
func (t *T) Fatalf(format string, args ...any) {
	fmt.Println(format, args)
	panic("fatal")
}
func (t *T) Helper() {
	panic(unsupportedApi("t.Failed()"))
}
func (t *T) Log(args ...any) {
	fmt.Println(args...)
}

func (t *T) Logf(format string, args ...any) {
	fmt.Println(format)
	fmt.Println(args...)
}

func (t *T) Name() string {
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

func (t *T) Skip(args ...any) {
	panic(unsupportedApi("t.Skip()"))
}
func (t *T) SkipNow() {
	panic(unsupportedApi("t.SkipNow()"))
}
func (t *T) Skipf(format string, args ...any) {
	panic(unsupportedApi("t.Skipf()"))
}
func (t *T) Skipped() bool {
	panic(unsupportedApi("t.Skipped()"))
}

func (t *T) TempDir() string {
	f, err := os.CreateTemp("/tmp", "fuzzdir-")
	if err != nil {
		panic(err)
	}
	t.CreatedDirs = append(t.CreatedDirs, f.Name())
	return f.Name()
}
