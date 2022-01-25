package utils

import (
	"fmt"
	"reflect"
	"testing"
)

func fuzzRune(f *F) {
	f.Fuzz(func(t *testing.T, input rune) {		
		switch reflect.TypeOf(input).String() {
		case "int32":
			fmt.Println("We created a rune")
		default:
			t.Fatalf("input is not rune but should be")
		}
		if string(input)!="A" {
			fmt.Println(string(input))
			t.Fatalf("Should be A")
		}
	})
}

func fuzzString(f *F) {
	f.Fuzz(func(t *testing.T, input string) {	
		switch reflect.TypeOf(input).String() {
		case "string":
			fmt.Println("We created a string")
		default:
			t.Fatalf("input is not string but should be")
		}
		if input!="ABC" {
			t.Fatalf("input should be 'ABC'")
		}
	})
}

func TestFuzzRune(t *testing.T) {
	data := []byte{0x41, 0x42, 0x43}
	fuzzer := &F{Data:data, T:t}
	fuzzRune(fuzzer)
}

func TestFuzzString(t *testing.T) {
	data := []byte{0x3, 0x41, 0x42, 0x43}
	fuzzer := &F{Data:data, T:t}
	fuzzString(fuzzer)
}