package input

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestIntReader(t *testing.T) {
	for i, tc := range []struct {
		input reflect.Kind
		want  string
	}{
		{reflect.Int8, "-86"},
		{reflect.Int16, "-21999"},
		{reflect.Int32, "-1441717709"},
		{reflect.Int64, "-6192130409072597385"},
		{reflect.Int, "-6192130409072597385"},
	} {
		s := NewSource([]byte{0xaa, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x01, 0x02, 0x03})
		have := fmt.Sprintf("%v", s.readInt(tc.input))
		if have != tc.want {
			t.Errorf("test %d: have %q want %q", i, have, tc.want)
		}
	}
}

func TestUintReader(t *testing.T) {
	for i, tc := range []struct {
		input reflect.Kind
		want  string
	}{
		{reflect.Uint8, "170"},
		{reflect.Uint16, "43537"},
		{reflect.Uint32, "2853249587"},
		{reflect.Uint64, "12254613664636954231"},
		{reflect.Uint, "12254613664636954231"},
	} {
		s := NewSource([]byte{0xaa, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x01, 0x02, 0x03})
		have := fmt.Sprintf("%v", s.readUint(tc.input))
		if have != tc.want {
			t.Errorf("test %d: have %q want %q", i, have, tc.want)
		}
	}
}

func TestInputMatcher(t *testing.T) {
	var have string = "not invoked"

	fuzzFunc := func(t *testing.T,
		a uint, b uint8, c uint16, d uint32, e uint64,
		f int, g int8, h int16, i int32, j int64,
		k float32, l float64,
		o string, // eats all input
		m bool,
		n rune, // rune is only an alias for int32
	) {
		have = fmt.Sprint(a, b, c, d, e,
			f, g, h, i, j,
			k, l, m, n, o)
	}
	NewSource(fibonacci(80)).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
	want := "282583128934413 21 8759 1502669177 7123354410338337327 -1071837808048229080 -35 1506 -406212487 3000183971744439682 3.2462019e+19 5.311042028320797e+161 true 969935321\xbb\x9dX\xf5MB\x8f\xd1`1\x91\xc2S\x15h}"
	if have != want {
		t.Fatalf("result wrong\nhave %q\nwant %q", have, want)
	}
}
func TestInputMatcher2(t *testing.T) {
	var have string = "not invoked"

	fuzzFunc := func(t *testing.T,
		a uint, b uint8, c uint16, d uint32, e uint64,
		o []byte, // eats all input
		f int, g int8, h int16, i int32, j int64,
		k float32, l float64,
		m bool,
		n rune, // rune is only an alias for int32
	) {
		have = fmt.Sprint(a, b, c, d, e,
			f, g, h, i, j,
			k, l, m, n, string(o))
	}
	NewSource(fibonacci(80)).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
	want := "282583128934413 21 8759 1502669177 7123354410338337327 -1071837808048229080 -35 1506 -406212487 3000183971744439682 3.2462019e+19 5.311042028320797e+161 true 969935321\xbb\x9dX\xf5MB\x8f\xd1`1\x91\xc2S\x15h}"
	if have != want {
		t.Fatalf("result wrong\nhave %q\nwant %q", have, want)
	}
}

func fibonacci(size int) []byte {
	data := make([]byte, size)
	data[1] = 1
	for i := 2; i < len(data); i++ {
		data[i] = data[i-1] + data[i-2]
	}
	return data
}

func TestInputMatcher3(t *testing.T) {
	var have string = "not invoked"

	fuzzFunc := func(t *testing.T,
		a uint, b uint8, c uint16, d uint32, e uint64,
		s1 string, // eats all input
		s2 string, // eats all input
		f int, g int8, h int16, i int32, j int64,
		k float32, l float64, m bool,
	) {
		have = fmt.Sprint(a, b, c, d, e, f, g, h, i, j, k, l, m, s1, "|", s2)
	}
	input := bytes.NewBuffer(nil)
	binary.Write(input, binary.BigEndian, uint64(1_000_000_000))
	binary.Write(input, binary.BigEndian, uint8(100))
	binary.Write(input, binary.BigEndian, uint16(15000))
	binary.Write(input, binary.BigEndian, uint32(math.MaxUint32))
	binary.Write(input, binary.BigEndian, uint64(1_000_000_000_000_000))

	binary.Write(input, binary.BigEndian, int64(-1_000_000_000))
	binary.Write(input, binary.BigEndian, int8(-100))
	binary.Write(input, binary.BigEndian, int16(-15000))
	binary.Write(input, binary.BigEndian, int32(-math.MaxInt32))
	binary.Write(input, binary.BigEndian, int64(-1_000_000_000_000_000))

	binary.Write(input, binary.BigEndian, float32(3.14159265358979323846))
	binary.Write(input, binary.BigEndian, float64(3.14159265358979323846))
	binary.Write(input, binary.BigEndian, true)

	input.Write([]byte{100, 100})
	input.WriteString("choo-choobloo-bloo")

	NewSource(input.Bytes()).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
	want := "1000000000 100 15000 4294967295 1000000000000000 -1000000000 -100 -15000 -2147483647 -1000000000000000 3.1415927 3.141592653589793 truechoo-choo|bloo-bloo"
	if have != want {
		t.Fatalf("result wrong\nhave %q\nwant %q", have, want)
	}
}

func TestDynamicArgs(t *testing.T) {
	var have string = "not invoked"
	fuzzFunc := func(t *testing.T, s1, s2, s3, s4 string) {
		have = fmt.Sprint(s1, "|", s2, "|", s3, "|", s4)
	}
	input := bytes.NewBuffer(nil)
	input.Write([]byte{1, 10, 5, 5})
	input.WriteString("122222222223333344444")

	NewSource(input.Bytes()).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
	want := "1|2222222222|33333|44444"
	if have != want {
		t.Fatalf("result wrong\nhave %q\nwant %q", have, want)
	}
}

func TestDynamicArgsZeroWeight(t *testing.T) {
	var have string = "not invoked"
	fuzzFunc := func(t *testing.T, s1, s2, s3, s4, s5 string) {
		have = fmt.Sprint(s1, "|", s2, "|", s3, "|", s4, "|", s5)
	}
	input := bytes.NewBuffer(nil)
	input.Write([]byte{0, 0, 0, 0, 0})
	input.WriteString("11112222333344445555")

	NewSource(input.Bytes()).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
	want := "1111|2222|3333|4444|5555"
	if have != want {
		t.Fatalf("result wrong\nhave %q\nwant %q", have, want)
	}
}

func TestExhausted(t *testing.T) {
	input := NewSource(make([]byte, 8))
	input.fillArg(reflect.TypeOf(uint64(0)), 0)          // Consumes 8 byte
	input.fillArg(reflect.TypeOf([]byte{}), input.Len()) // Consumes nothing
	input.fillArg(reflect.TypeOf(""), input.Len())       // Consumes nothing
	if input.IsExhausted() {
		t.Fatalf("expected not exhausted")
	}
	input.fillArg(reflect.TypeOf(uint8(0)), 0) // Consumes 1 byte
	if !input.IsExhausted() {
		t.Fatalf("expected exhausted")
	}
}

func TestReader(t *testing.T) {

	{
		s := NewSource(fibonacci(100))
		s.getBytes(100)
		if s.IsExhausted() {
			t.Fatal("exp not exhausted")
		}
	}
	{
		s := NewSource(fibonacci(100))
		s.getBytes(101)
		if !s.IsExhausted() {
			t.Fatal("exp exhausted")
		}
	}
	{
		s := NewSource(fibonacci(100))
		s.getBytes(100)
		s.getBytes(0)
		s.getBytes(0)
		if s.IsExhausted() {
			t.Fatal("exp not exhausted")
		}
	}
}