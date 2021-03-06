package utils

import (
	//"fmt"
	"reflect"
	"testing"
	//fuzz "github.com/AdaLogics/go-fuzz-+"
)

func fuzzRune(f *F) {
	f.Fuzz(func(t *testing.T, input rune) {
		if reflect.TypeOf(input).String() != "int32" {
			t.Fatalf("input is not int but should be")
		}
		if string(input) != "A" {
			t.Fatalf("Should be A")
		}
	})
}

func fuzzString(f *F) {
	f.Fuzz(func(t *testing.T, input string) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input != "ABC" {
			t.Fatalf("input should be 'ABC'")
		}
	})
}

func fuzzTwoStrings(f *F) {
	f.Fuzz(func(t *testing.T, input, input2 string) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input != "AB" {
			t.Fatalf("input should be 'AB' but is %s\n", input)
		}
		if reflect.TypeOf(input2).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input2 != "C" {
			t.Fatalf("input should be 'C' but is %s\n", input2)
		}
	})
}

func fuzzThreeArgs(f *F) {
	f.Fuzz(func(t *testing.T, input, input2 string, input3 int) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input != "AB" {
			t.Fatalf("input should be 'AB' but is %s\n", input)
		}
		if reflect.TypeOf(input2).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input2 != "C" {
			t.Fatalf("input should be 'C' but is %s\n", input2)
		}
		if reflect.TypeOf(input3).String() != "int" {
			t.Fatalf("input is not int but should be")
		}
		if input3 != 68 {
			t.Fatalf("input3 should be '68'")
		}
	})
}

func fuzzFourArgs(f *F) {
	f.Fuzz(func(t *testing.T, input, input2 string, input3 int, input4 uint32) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input != "AB" {
			t.Fatalf("input should be 'AB' but is %s\n", input)
		}
		if reflect.TypeOf(input2).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input2 != "C" {
			t.Fatalf("input should be 'C' but is %s\n", input2)
		}
		if reflect.TypeOf(input3).String() != "int" {
			t.Fatalf("input is not int but should be")
		}
		if input3 != 68 {
			t.Fatalf("input3 should be '68'")
		}
		if reflect.TypeOf(input4).String() != "uint32" {
			t.Fatalf("input is not uint32 but should be")
		}
		if input4 != 1162233672 {
			t.Fatalf("input3 should be '1162233672'")
		}

	})
}

func fuzzFloat32(f *F) {
	f.Fuzz(func(t *testing.T, input float32) {
		if reflect.TypeOf(input).String() != "float32" {
			t.Fatalf("input is not float32 but should be")
		}
		expectedFloat32 := float32(194.25395)
		if input != expectedFloat32 {
			t.Errorf("'newFloat' should be '%f', but is %f\n", expectedFloat32, input)
		}
	})
}

func fuzzFloat64(f *F) {
	f.Fuzz(func(t *testing.T, input float64) {
		if reflect.TypeOf(input).String() != "float64" {
			t.Fatalf("input is not float64 but should be")
		}
		expectedFloat64 := 2.3127085096212183e+35
		if input != expectedFloat64 {
			t.Errorf("'newFloat' should be '%f', but is %f\n", expectedFloat64, input)
		}
	})
}

func TestFuzzFloat32(t *testing.T) {
	data := []byte{0x3, 0x41, 0x42, 0x43, 0x44}
	f := &F{Data: data, T: t}
	fuzzFloat32(f)
}

func TestFuzzFloat64(t *testing.T) {
	data := []byte{0x3, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48}
	f := &F{Data: data, T: t}
	fuzzFloat64(f)
}

func TestFuzzRune(t *testing.T) {
	data := []byte{0x41, 0x42, 0x43}
	f := &F{Data: data, T: t}
	fuzzRune(f)
}

func TestFuzzString(t *testing.T) {
	data := []byte{0x3, 0x41, 0x42, 0x43}
	f := &F{Data: data, T: t}
	fuzzString(f)
}

func TestFuzzTwoStrings(t *testing.T) {
	data := []byte{0x2, 0x41, 0x42, 0x1, 0x43}
	f := &F{Data: data, T: t}
	fuzzTwoStrings(f)
}

func TestFuzzFourArgs(t *testing.T) {
	data := []byte{0x2, 0x41, 0x42, 0x1, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49}
	f := &F{Data: data, T: t}
	fuzzFourArgs(f)
}
