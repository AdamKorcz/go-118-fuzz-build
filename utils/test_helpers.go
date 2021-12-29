package utils

import (
	"fmt"
	"reflect"
	"testing"
)

type F struct {
	Data []byte
	T *testing.T
	FuzzFunc func(*testing.T, any)
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
func (f *F) Fuzz(ff any) {
	fn := reflect.ValueOf(ff)
	fnType := fn.Type()
	var types []reflect.Type
	for i := 1; i < fnType.NumIn(); i++ {
		t := fnType.In(i)
		types = append(types, t)
	}
	args := []reflect.Value{reflect.ValueOf(f.T)}
	for _, v := range types {
		fmt.Println(v.Kind())
		args = append(args, reflect.ValueOf(v))
	}
	fmt.Println(args)
	//ff(f.T, params)
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
