package utils

import (
	//"fmt"
	"reflect"
	"testing"
	//fuzz "github.com/AdaLogics/go-fuzz-headers"
)

func fuzzRune(f *F) {
	f.Fuzz(func(t *testing.T, input rune) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if string(input)!="A" {
			t.Fatalf("Should be A")
		}
	})
}

func fuzzString(f *F) {
	f.Fuzz(func(t *testing.T, input string) {	
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input!="ABC" {
			t.Fatalf("input should be 'ABC'")
		}
	})
}

func fuzzTwoStrings(f *F) {
	f.Fuzz(func(t *testing.T, input, input2 string) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input!="AB" {
			t.Fatalf("input should be 'AB' but is %s\n", input)
		}
		if reflect.TypeOf(input2).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input2!="C" {
			t.Fatalf("input should be 'C' but is %s\n", input2)
		}
	})
}

func fuzzThreeArgs(f *F) {
	f.Fuzz(func(t *testing.T, input, input2 string, input3 int) {
		if reflect.TypeOf(input).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input!="AB" {
			t.Fatalf("input should be 'AB' but is %s\n", input)
		}
		if reflect.TypeOf(input2).String() != "string" {
			t.Fatalf("input is not string but should be")
		}
		if input2!="C" {
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

func TestFuzzRune(t *testing.T) {
	data := []byte{0x41, 0x42, 0x43}
	f := &F{Data:data, T:t}
	fuzzRune(f)
}

func TestFuzzString(t *testing.T) {
	data := []byte{0x3, 0x41, 0x42, 0x43}
	f := &F{Data:data, T:t}
	fuzzString(f)
}

func TestFuzzTwoStrings(t *testing.T) {
	data := []byte{0x2, 0x41, 0x42, 0x1, 0x43}
	f := &F{Data:data, T:t}
	fuzzTwoStrings(f)
}

func TestFuzzThreeArgs(t *testing.T) {	
	data := []byte{0x2, 0x41, 0x42, 0x1, 0x43, 0x44}
	f := &F{Data:data, T:t}
	fuzzThreeArgs(f)
}