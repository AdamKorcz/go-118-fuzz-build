package coverage

import (
	//"fmt"
	"testing"
)

var (
	libFuzzertestCase1 = []byte{5, 65, 68, 65, 77, 49, 4, 65, 68, 65, 77, 1, 2, 0, 65, 66, 65, 77, 49, 0, 0, 4}
	goTestCase1        = `go test fuzz v1
string("ADAM1")
string("ADAM")
int(1)
[]byte("AB")`
	fuzzer1 = `package fuzzpackage

import (
	"fmt"
	"testing"
)

func FuzzTest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data1, data2 string, in int, data3 []byte) {
		fmt.Println("HERE")
	})
}`
)

func TestGetFuzzArgs(t *testing.T) {
	args, err := getFuzzArgs(fuzzer1)
	if err != nil {
		t.Error(err)
	}
	if args[0] != "string" {
		t.Log(fuzzer1)
		t.Error("args[0] should be []byte but is not")
	}
	if args[1] != "string" {
		t.Log(fuzzer1)
		t.Error("args[0] should be []byte but is not")
	}
	if args[2] != "int" {
		t.Log(fuzzer1)
		t.Error("args[0] should be int but is not")
	}
	if args[3] != "[]byte" {
		t.Log(fuzzer1)
		t.Error("args[0] should be string but is not")
	}

	libFuzzerTestcase := libFuzzerSeedToGoSeed(libFuzzertestCase1, args)
	if libFuzzerTestcase != goTestCase1 {
		t.Error("Failed testcase conversion")
	}
}
