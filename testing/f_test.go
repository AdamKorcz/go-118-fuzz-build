package testing

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"
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

func TestFuzz(t *testing.T) {
	var have string = "not invoked"

	fuzzFunc := func(t *T,
		a int, b int8, c int16, d int32, e int64,
		f uint, g uint8, h uint16, i uint32, j uint64,
		k string, l []byte,
		m float64, n float32, o bool, p rune) {
		have = fmt.Sprint(a, b, c, d, e, f, g, h, i, j, m, n, o, p)
	}

	f := new(F)
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

	f.Data = input
	f.Fuzz(fuzzFunc)
	want := "-1 -1 -1 -1 -1 1234605616150177399 17 8755 1146447479 1234605616150177399 1.1337 3.14159 true 3"
	if have != want {
		t.Fatalf("result wrong\nhave %q\nwant %q", have, want)
	}
}
