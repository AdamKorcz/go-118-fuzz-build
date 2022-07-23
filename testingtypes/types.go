package testingtypes

import (
	"fmt"
	"strings"
	"time"
)

type T struct {
}

func unsupportedApi(name string) string {
	plsOpenIss := "Please open an issue https://github.com/AdamKorcz/go-118-fuzz-build if you need this feature."
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s is not supported when fuzzing in libFuzzer mode\n.", name))
	b.WriteString(plsOpenIss)
	return b.String()
}

func Cleanup(f func()) {
	f()
}
func Deadline() (deadline time.Time, ok bool) {
	panic(unsupportedApi("t.Deadline()"))
}

func Error(args ...any) {
	fmt.Println(args...)
	panic("error")
}

func Errorf(format string, args ...any) {
	fmt.Println(format)
	fmt.Println(args...)
	panic("errorf")
}

func Fail() {
	panic(unsupportedApi("t.Fail()"))
}

func FailNow() {
	panic(unsupportedApi("t.FailNow()"))
}

func Failed() bool {
	panic(unsupportedApi("t.Failed()"))
}

func Fatal(args ...any) {
	fmt.Println(args...)
	panic("fatal")
}
func Fatalf(format string, args ...any) {
	fmt.Println(format, args)
	panic("fatal")
}
func Helper() {
	panic(unsupportedApi("t.Failed()"))
}
func Log(args ...any) {
	fmt.Println(args...)
}

func Logf(format string, args ...any) {
	fmt.Println(format)
	fmt.Println(args...)
}

func Name() string {
	return "fuzzer"
}

func Parallel() {
	panic(unsupportedApi("t.Failed()"))
}
func Run(name string, f func(t *T)) bool {
	panic(unsupportedApi("t.Run()."))
}

func Setenv(key, value string) {

}

func Skip(args ...any) {
	panic(unsupportedApi("t.Skip()"))
}
func SkipNow() {
	panic(unsupportedApi("t.SkipNow()"))
}
func Skipf(format string, args ...any) {
	panic(unsupportedApi("t.Skipf()"))
}
func Skipped() bool {
	panic(unsupportedApi("t.Skipped()"))
}
func TempDir() string {
	panic(unsupportedApi("t.TempDir()"))
}
