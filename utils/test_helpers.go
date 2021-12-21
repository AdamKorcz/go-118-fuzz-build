package utils

import (
	"testing"
)

type F struct {
	Data []byte
	T *testing.T
}
func (f *F) Add(args ...any) {}
func (c *F) Cleanup(f func()) {}
func (c *F) Error(args ...any) {}
func (c *F) Errorf(format string, args ...any) {}
func (f *F) Fail() {}
func (c *F) FailNow() {}
func (c *F) Failed() bool {return false}
func (c *F) Fatal(args ...any) {}
func (c *F) Fatalf(format string, args ...any) {}
func (f *F) Fuzz(ff func(t *testing.T, data []byte)) {
	ff(f.T, f.Data)
}
func (f *F) Helper() {}
func (c *F) Log(args ...any) {}
func (c *F) Logf(format string, args ...any) {}
func (c *F) Name() string {return "name"}
func (c *F) Setenv(key, value string) {}
func (c *F) Skip(args ...any) {}
func (c *F) SkipNow() {}
func (c *F) Skipf(format string, args ...any) {}
func (f *F) Skipped() bool {return false}
func (c *F) TempDir() string {return "/tmp"}
