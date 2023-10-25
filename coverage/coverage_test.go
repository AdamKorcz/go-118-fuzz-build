package coverage

import (
	"encoding/binary"
	"testing"
)

var (
	goTestCase1 = `go test fuzz v1
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

// addString adds a string to the input vector corresponding to consumer.go @ go-fuzz-headers
func addString(input []byte, s string) []byte {
	input = binary.BigEndian.AppendUint32(input, uint32(len(s))) // Add a uint32 length
	input = append(input, []byte(s)...)                          // Add string
	return input
}

// addString adds a []byte to the input vector corresponding to consumer.go @ go-fuzz-headers
func addBytes(input []byte, data []byte) []byte {
	input = binary.BigEndian.AppendUint32(input, uint32(len(data))) // Add a uint32 length
	input = append(input, data...)                                  // Add string
	return input
}

// addString adds a uint64 to the input vector corresponding to consumer.go @ go-fuzz-headers
func addU64(input []byte, i uint64) []byte {
	input = binary.BigEndian.AppendUint64(input, uint64(i))
	input = append(input, 1) // endianness boolean
	return input
}

func TestGetFuzzArgs(t *testing.T) {
	args, err := getFuzzArgs(fuzzer1, "FuzzTest")
	if err != nil {
		t.Error(err)
	}
	for i, want := range []string{"string", "string", "int", "[]byte"} {
		if have := args[i]; have != want {
			t.Errorf("args[%d] wrong: have %q want %q", i, have, want)
		}
	}
	testdata := addString(nil, "ADAM1")
	testdata = addString(testdata, "ADAM")
	testdata = addU64(testdata, 1)
	testdata = addBytes(testdata, []byte("AB"))

	have := libFuzzerSeedToGoSeed(testdata, args)
	if want := goTestCase1; have != want {
		t.Logf("have\n%v\nwant\n%v\n", have, want)
		t.Error("Failed testcase conversion")
	}
}
