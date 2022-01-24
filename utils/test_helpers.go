package utils

import (
	"fmt"
	"reflect"
	"testing"
	fuzz "github.com/AdaLogics/go-fuzz-headers"
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
	// we are assuming that ff is a func.
	// TODO: Add a check for UX purposes

	fn := reflect.ValueOf(ff)
	fnType := fn.Type()
	var types []reflect.Type
	for i := 1; i < fnType.NumIn(); i++ {
		t := fnType.In(i)

		types = append(types, t)
	}
	args := []reflect.Value{reflect.ValueOf(f.T)}
	fuzzConsumer := fuzz.NewConsumer(f.Data)
	for _, v := range types {
		switch v.String() {
		case "[]uint8":
			b, err := fuzzConsumer.GetBytes()
			if err != nil {
				return
			}
			newBytes := reflect.New(v)
			newBytes.Elem().SetBytes(b)
			args = append(args, newBytes.Elem())
		case "string":
			s, err := fuzzConsumer.GetString()
			if err != nil {
				return
			}
			newString := reflect.New(v)
			newString.Elem().SetString(s)
			args = append(args, newString.Elem())
		case "int":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(randInt)
			args = append(args, newInt.Elem())
		case "int8":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int8(randInt))
			args = append(args, newInt.Elem())
		case "int16":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int16(randInt))
			args = append(args, newInt.Elem())
		case "int32":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int32(randInt))
			args = append(args, newInt.Elem())
		case "int64":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int64(randInt))
			args = append(args, newInt.Elem())
		case "uint":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newUint := reflect.New(v)
			newUint.Elem().SetUint(uint(randInt))
			args = append(args, newUint.Elem())
		case "uint8":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				return
			}
			newUint := reflect.New(v)
			newUint.Elem().SetUint(uint8(randInt))
			args = append(args, newUint.Elem())
		case "uint16":
			randInt, err := fuzzConsumer.GetUin16()
			if err != nil {
				return
			}
			newUint16 := reflect.New(v)
			newUint16.Elem().SetUint(randInt)
			args = append(args, newUint16.Elem())
		case "uint32":
			randInt, err := fuzzConsumer.GetUin32()
			if err != nil {
				return
			}
			newUint32 := reflect.New(v)
			newUint32.Elem().SetUint(randInt)
			args = append(args, newUint32.Elem())
		case "uint64":
			randInt, err := fuzzConsumer.GetUint64()
			if err != nil {
				return
			}
			newUint64 := reflect.New(v)
			newUint64.Elem().SetUint(uint64(randInt))
			args = append(args, newUint64.Elem())
		default: 
			fmt.Println(v.String())
		}
	}
	fn.Call(args)
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
