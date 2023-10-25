package coverage

import (
	"encoding/binary"
	"math"
	"testing"
)

var (
	fuzzer1 = `package fuzzpackage

import (
	"fmt"
	"testing"
)

func FuzzTest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data1, data2 string, in int, data3 []byte) {
		fmt.Println("HERE")
	})
}

func FuzzTest2(f *testing.F) {
	f.Fuzz(func(t *testing.T, 
		a int, b int8, c int16, d int32, e int64, 
		f uint, g uint8, h uint16, i uint32, j uint64,
		foo string, bar []byte,
		f1 float64, f2 float32, bb bool, rr rune){
		fmt.Println("HERE")
	})
}
`
)

// addString adds a string to the input vector corresponding to consumer.go @ go-fuzz-headers
func addString(input []byte, s string) []byte {
	input = binary.BigEndian.AppendUint32(input, uint32(len(s))) // Add a uint32 length
	input = append(input, []byte(s)...)                          // Add string
	return input
}

// addBytes adds a []byte to the input vector corresponding to consumer.go @ go-fuzz-headers
func addBytes(input []byte, data []byte) []byte {
	input = binary.BigEndian.AppendUint32(input, uint32(len(data))) // Add a uint32 length
	input = append(input, data...)                                  // Add string
	return input
}

// addU64 adds a uint64 to the input vector corresponding to consumer.go @ go-fuzz-headers
func addU64(input []byte, i uint64) []byte {
	input = binary.BigEndian.AppendUint64(input, i)
	input = append(input, 1) // endianness boolean
	return input
}

func addU16(input []byte, i uint64) []byte {
	input = binary.BigEndian.AppendUint16(input, uint16(i))
	input = append(input, 1) // endianness boolean
	return input
}

func addU32(input []byte, i uint64) []byte {
	input = binary.BigEndian.AppendUint32(input, uint32(i))
	// U32 doesn't use endianness boolean! (but when used for float32, it does, sigh)
	return input
}

func addF32(input []byte, f float32) []byte {
	input = binary.BigEndian.AppendUint32(input, uint32(math.Float32bits(f)))
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
	input := addString(nil, "ADAM1")
	input = addString(input, "ADAM")
	input = addU64(input, 1)
	input = addBytes(input, []byte("AB"))

	have := libFuzzerSeedToGoSeed(input, args)
	want := `go test fuzz v1
string("ADAM1")
string("ADAM")
int(1)
[]byte("AB")`

	if have != want {
		t.Logf("have\n%v\nwant\n%v\n", have, want)
		t.Error("Failed testcase conversion")
	}
}

func TestGetFuzzArgs2(t *testing.T) {
	args, err := getFuzzArgs(fuzzer1, "FuzzTest2")
	if err != nil {
		t.Error(err)
	}
	for i, want := range []string{"int", "int8", "int16", "int32", "int64", "uint",
		"uint8", "uint16", "uint32", "uint64", "string", "[]byte", "float64", "float32",
		"bool", "rune"} {
		if have := args[i]; have != want {
			t.Errorf("args[%d] wrong: have %q want %q", i, have, want)
		}
	}
	var v int64 = -1
	input := addU64(nil, uint64(v))         // int
	input = append(input, byte(int8(v)))    // int8
	input = addU16(input, uint64(int16(v))) // int16
	input = addU32(input, uint64(int32(v))) // int32
	input = addU64(input, uint64(int64(v))) // int64

	input = addU64(input, uint64(0x1122_3344_4455_6677)) // uint
	input = append(input, 0x11)                          // uint8
	input = addU16(input, uint64(uint16(0x2233)))        // uint16
	input = addU32(input, uint64(uint32(0x4455_6677)))   // uint32
	input = addU64(input, uint64(0x1122_3344_4455_6677)) // uint64

	input = addString(input, "string\x00oll\nkorrekt")
	input = addBytes(input, []byte("bytes\x00oll\nkorrekt"))

	input = addU64(input, math.Float64bits(1.1337)) // float64
	input = addF32(input, float32(3.14159))         // float32

	input = append(input, 0) // boolean true
	// Note: the fuzzer doesn't treat runes correctly, instead of a rune being a
	// rune, it treats them as strings.
	input = addString(input, string([]rune{rune('Ⅷ')})) // rune

	have := libFuzzerSeedToGoSeed(input, args)
	want := `go test fuzz v1
int(-1)
int8(-1)
int16(-1)
int32(-1)
int64(-1)
uint(1234605616150177399)
uint8(17)
uint16(8755)
uint32(1146447479)
uint64(1234605616150177399)
string("string\x00oll\nkorrekt")
[]byte("bytes\x00oll\nkorrekt")
float64(1.133700)
float32(3.141590)
bool(true)
rune("Ⅷ")`

	if have != want {
		t.Logf("have\n%v\nwant\n%v\n", have, want)
		t.Error("Failed testcase conversion")
	}
}
