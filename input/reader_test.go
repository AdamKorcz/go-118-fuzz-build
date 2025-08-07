package input

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
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

func TestCorpusConversion(t *testing.T) {
	var have string

	for _, tc := range []struct {
		want         string
		data         string
		wantTestcase string
		fuzzFunc     func(t *testing.T, a, b string, c []byte, d, e int, f uint32, g uint64)
	}{
		{
			want: fmt.Sprint("EVE", "NEIG", []byte("HTNINE"), int(4702111238803703110), int(5714581205724124232), uint32(1380271430), uint64(5716565763848291667)),
			data: "AAABCDEFONETWOTHREEFOURFIVESIXSEVENEIGHTNINE",
			wantTestcase: `go test fuzz v1
string("EVE")
string("NEIG")
[]byte("HTNINE")
int(4702111238803703110)
int(5714581205724124232)
uint32(1380271430)
uint64(5716565763848291667)`,
			fuzzFunc: func(t *testing.T, a, b string, c []byte, d, e int, f uint32, g uint64) {
				have = fmt.Sprint(a, b, c, d, e, f, g)
			},
		},
		{
			want: fmt.Sprint("AAAA", "AAAA", []byte("AAAAAA"), int(5714581123968029783), int(5710931857861268037), uint32(1161907780), uint64(5066361917585572161)),
			data: "ONEANDTWOANDTHREEANDFOURAAAAAAAAAAAAAAAAAAAAA",
			wantTestcase: `go test fuzz v1
string("AAAA")
string("AAAA")
[]byte("AAAAAA")
int(5714581123968029783)
int(5710931857861268037)
uint32(1161907780)
uint64(5066361917585572161)`,
			fuzzFunc: func(t *testing.T, a, b string, c []byte, d, e int, f uint32, g uint64) {
				have = fmt.Sprint(a, b, c, d, e, f, g)
			},
		},
		{
			want: fmt.Sprint("AAAA", "AAAA", []byte("BBBBBB"), int(5714581123968029783), int(5710931857861268037), uint32(1161907780), uint64(5066361917585572161)),
			data: "ONEANDTWOANDTHREEANDFOURAAAAAAAAAAAAAAABBBBBB",
			wantTestcase: `go test fuzz v1
string("AAAA")
string("AAAA")
[]byte("BBBBBB")
int(5714581123968029783)
int(5710931857861268037)
uint32(1161907780)
uint64(5066361917585572161)`,
			fuzzFunc: func(t *testing.T, a, b string, c []byte, d, e int, f uint32, g uint64) {
				have = fmt.Sprint(a, b, c, d, e, f, g)
			},
		},
	} {
		have = "not invoked"
		s := NewSource([]byte(tc.data))
		haveTestcase := s.CreateGoTestcase(tc.fuzzFunc, reflect.ValueOf(new(testing.T)))
		if haveTestcase != tc.wantTestcase {
			t.Errorf("Created wrong testcase:got: '%q' want: '%q'", haveTestcase, tc.wantTestcase)
		}
		NewSource([]byte(tc.data)).FillAndCall(tc.fuzzFunc, reflect.ValueOf(new(testing.T)))

		if have != tc.want {
			t.Errorf("Created wrong testcase: have:'%q' want: '%q'", have, tc.want)
		}
	}
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

func TestParseGoTestcase_Table(t *testing.T) {
	type TestCase struct {
		name     string
		testcase string
		fuzzFunc any
		expected func(t *testing.T)
	}

	var tests = []TestCase{
		{
			name: "Mixed int and string",
			testcase: `go test fuzz v1
int8(42)
string("hello")`,
			fuzzFunc: func(t *testing.T, a int8, b string) {
				if a != 42 {
					t.Errorf("a = %d; want 42", a)
				}
				if b != "hello" {
					t.Errorf("b = %q; want hello", b)
				}
			},
		},
		{
			name: "Unicode string",
			testcase: `go test fuzz v1
string("こんにちは")`,
			fuzzFunc: func(t *testing.T, s string) {
				if s != "こんにちは" {
					t.Errorf("s = %q; want こんにちは", s)
				}
			},
		},
		{
			name: "Mixed uint and []byte",
			testcase: `go test fuzz v1
uint16(65535)
[]byte("üßñ")`,
			fuzzFunc: func(t *testing.T, a uint16, b []byte) {
				if a != 65535 {
					t.Errorf("a = %d; want 65535", a)
				}
				if string(b) != "üßñ" {
					t.Errorf("b = %q; want üßñ", b)
				}
			},
		},
		{
			name: "Float32, Float64, and bool",
			testcase: `go test fuzz v1
float32(1.5)
float64(3.14159)
bool(true)`,
			fuzzFunc: func(t *testing.T, f1 float32, f2 float64, b bool) {
				if math.Abs(float64(f1)-1.5) > 0.001 {
					t.Errorf("f1 = %f; want 1.5", f1)
				}
				if math.Abs(f2-3.14159) > 0.00001 {
					t.Errorf("f2 = %f; want 3.14159", f2)
				}
				if b != true {
					t.Errorf("b = %v; want true", b)
				}
			},
		},
		{
			name: "int, uint, bool, string",
			testcase: `go test fuzz v1
int(42)
uint(99)
bool(false)
string("ß")`,
			fuzzFunc: func(t *testing.T, a int, b uint, c bool, d string) {
				if a != 42 || b != 99 || c != false || d != "ß" {
					t.Errorf("Got: %v %v %v %q", a, b, c, d)
				}
			},
		},
		{
			name: "Empty string and bytes",
			testcase: `go test fuzz v1
string("")
[]byte("")`,
			fuzzFunc: func(t *testing.T, s string, b []byte) {
				if s != "" || len(b) != 0 {
					t.Errorf("Expected empty string and byte slice")
				}
			},
		},
		{
			name: "Multiple Unicode strings",
			testcase: `go test fuzz v1
string("💖")
string("🚀")
string("🌍")`,
			fuzzFunc: func(t *testing.T, a, b, c string) {
				if a != "💖" || b != "🚀" || c != "🌍" {
					t.Errorf("Got: %q %q %q", a, b, c)
				}
			},
		},
		{
			name: "Mix of all primitives",
			testcase: `go test fuzz v1
int32(-500)
uint64(1234567890123456)
float32(2.718)
float64(6.28)
bool(true)
string("中")`,
			fuzzFunc: func(t *testing.T, i int32, u uint64, f1 float32, f2 float64, b bool, s string) {
				if i != -500 || u != 1234567890123456 || s != "中" || !b {
					t.Errorf("Unexpected values: %v %v %v %v %v %q", i, u, f1, f2, b, s)
				}
			},
		},
		{
			name: "Bool false",
			testcase: `go test fuzz v1
bool(false)`,
			fuzzFunc: func(t *testing.T, b bool) {
				if b {
					t.Errorf("b = true; want false")
				}
			},
		},
		{
			name: "Edge values",
			testcase: `go test fuzz v1
int8(-128)
int8(127)
uint8(0)
uint8(255)`,
			fuzzFunc: func(t *testing.T, a, b int8, c, d uint8) {
				if a != -128 || b != 127 || c != 0 || d != 255 {
					t.Errorf("Edge mismatch: %d %d %d %d", a, b, c, d)
				}
			},
		},
		{
			name: "UTF-8 and extended bytes",
			testcase: `go test fuzz v1
[]byte("©π∆ß")`,
			fuzzFunc: func(t *testing.T, b []byte) {
				if string(b) != "©π∆ß" {
					t.Errorf("Got: %q", string(b))
				}
			},
		},
		{
			name: "Long Unicode",
			testcase: `go test fuzz v1
string("你好，世界")`,
			fuzzFunc: func(t *testing.T, s string) {
				if s != "你好，世界" {
					t.Errorf("Got: %q", s)
				}
			},
		},
		{
			name: "Emoji byte slice",
			testcase: `go test fuzz v1
[]byte("🧪📦🔬")`,
			fuzzFunc: func(t *testing.T, b []byte) {
				if string(b) != "🧪📦🔬" {
					t.Errorf("Got: %q", string(b))
				}
			},
		},
		{
			name: "Int and emoji string",
			testcase: `go test fuzz v1
int32(2048)
string("🦊")`,
			fuzzFunc: func(t *testing.T, i int32, s string) {
				if i != 2048 || s != "🦊" {
					t.Errorf("Got: %v %q", i, s)
				}
			},
		},
		{
			name: "Russian string",
			testcase: `go test fuzz v1
string("Привет")`,
			fuzzFunc: func(t *testing.T, s string) {
				if s != "Привет" {
					t.Errorf("Got: %q", s)
				}
			},
		},
		{
			name: "10 int32s",
			testcase: `go test fuzz v1
int32(1)
int32(2)
int32(3)
int32(4)
int32(5)
int32(6)
int32(7)
int32(8)
int32(9)
int32(10)`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j int32) {
				want := []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
				have := []int32{a, b, c, d, e, f, g, h, i, j}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("int32[%d] = %d; want %d", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "10 strings",
			testcase: `go test fuzz v1
string("a")
string("b")
string("c")
string("d")
string("e")
string("f")
string("g")
string("h")
string("i")
string("j")`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j string) {
				want := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
				have := []string{a, b, c, d, e, f, g, h, i, j}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("string[%d] = %q; want %q", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "10 mixed ints",
			testcase: `go test fuzz v1
int8(1)
int16(2)
int32(3)
int64(4)
int(5)
uint8(6)
uint16(7)
uint32(8)
uint64(9)
uint(10)`,
			fuzzFunc: func(t *testing.T, a int8, b int16, c int32, d int64, e int,
				f uint8, g uint16, h uint32, i uint64, j uint) {
				if a != 1 || b != 2 || c != 3 || d != 4 || e != 5 ||
					f != 6 || g != 7 || h != 8 || i != 9 || j != 10 {
					t.Errorf("Got: %v %v %v %v %v %v %v %v %v %v",
						a, b, c, d, e, f, g, h, i, j)
				}
			},
		},
		{
			name: "10 bools alternating",
			testcase: `go test fuzz v1
bool(true)
bool(false)
bool(true)
bool(false)
bool(true)
bool(false)
bool(true)
bool(false)
bool(true)
bool(false)`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j bool) {
				want := []bool{true, false, true, false, true, false, true, false, true, false}
				have := []bool{a, b, c, d, e, f, g, h, i, j}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("bool[%d] = %v; want %v", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "10 float values",
			testcase: `go test fuzz v1
float32(1.1)
float32(2.2)
float32(3.3)
float64(4.4)
float64(5.5)
float64(6.6)
float32(7.7)
float64(8.8)
float32(9.9)
float64(10.01)`,
			fuzzFunc: func(t *testing.T, a, b, c float32, d, e, f float64, g float32, h float64, i float32, j float64) {
				if a != 1.1 || b != 2.2 || c != 3.3 || d != 4.4 || e != 5.5 ||
					f != 6.6 || g != 7.7 || h != 8.8 || i != 9.9 || j != 10.01 {
					t.Errorf("Unexpected float values")
				}
			},
		},
		{
			name: "10 []byte values",
			testcase: `go test fuzz v1
[]byte("a")
[]byte("b")
[]byte("c")
[]byte("d")
[]byte("e")
[]byte("f")
[]byte("g")
[]byte("h")
[]byte("i")
[]byte("j")`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j []byte) {
				want := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
				have := []string{string(a), string(b), string(c), string(d), string(e),
					string(f), string(g), string(h), string(i), string(j)}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("[]byte[%d] = %q; want %q", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "10 emoji strings",
			testcase: `go test fuzz v1
string("😀")
string("😁")
string("😂")
string("🤣")
string("😃")
string("😄")
string("😅")
string("😆")
string("😉")
string("😊")`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j string) {
				want := []string{"😀", "😁", "😂", "🤣", "😃", "😄", "😅", "😆", "😉", "😊"}
				have := []string{a, b, c, d, e, f, g, h, i, j}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("emoji[%d] = %q; want %q", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "10 Unicode words",
			testcase: `go test fuzz v1
string("你好")
string("世界")
string("欢迎")
string("使用")
string("单元")
string("测试")
string("反射")
string("函数")
string("动态")
string("参数")`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j string) {
				want := []string{"你好", "世界", "欢迎", "使用", "单元", "测试", "反射", "函数", "动态", "参数"}
				have := []string{a, b, c, d, e, f, g, h, i, j}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("unicode[%d] = %q; want %q", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "10 alternating fixed+dynamic",
			testcase: `go test fuzz v1
int8(1)
string("a")
int16(2)
string("b")
int32(3)
string("c")
int64(4)
string("d")
int(5)
string("e")`,
			fuzzFunc: func(t *testing.T, a int8, b string, c int16, d string, e int32, f string, g int64, h string, i int, j string) {
				if a != 1 || b != "a" || c != 2 || d != "b" || e != 3 ||
					f != "c" || g != 4 || h != "d" || i != 5 || j != "e" {
					t.Errorf("Alternating values do not match")
				}
			},
		},
		{
			name: "10 dynamic strings with weights",
			testcase: `go test fuzz v1
string("111")
string("222")
string("333")
string("444")
string("555")
string("666")
string("777")
string("888")
string("999")
string("000")`,
			fuzzFunc: func(t *testing.T, a, b, c, d, e, f, g, h, i, j string) {
				want := []string{"111", "222", "333", "444", "555", "666", "777", "888", "999", "000"}
				have := []string{a, b, c, d, e, f, g, h, i, j}
				for i := range want {
					if want[i] != have[i] {
						t.Errorf("string[%d] = %q; want %q", i, have[i], want[i])
					}
				}
			},
		},
		{
			name: "All types 1",
			testcase: `go test fuzz v1
int8(-12)
uint16(65530)
float32(3.14)
bool(true)
string("abc")
[]byte("xyz")`,
			fuzzFunc: func(t *testing.T, a int8, b uint16, c float32, d bool, e string, f []byte) {
				if a != -12 || b != 65530 || c != 3.14 || !d || e != "abc" || string(f) != "xyz" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 2",
			testcase: `go test fuzz v1
int32(123456)
uint32(7890)
float64(6.28)
bool(false)
string("Ω≈ç√")
[]byte("†¥¨ˆøπ")`,
			fuzzFunc: func(t *testing.T, a int32, b uint32, c float64, d bool, e string, f []byte) {
				if a != 123456 || b != 7890 || c != 6.28 || d != false || e != "Ω≈ç√" || string(f) != "†¥¨ˆøπ" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 3",
			testcase: `go test fuzz v1
int(42)
uint(999)
float32(1.23)
float64(9.87)
bool(true)
string("hello")`,
			fuzzFunc: func(t *testing.T, a int, b uint, c float32, d float64, e bool, f string) {
				if a != 42 || b != 999 || c != 1.23 || d != 9.87 || !e || f != "hello" {
					t.Errorf("Mismatch: %v %v %v %v %v %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 4",
			testcase: `go test fuzz v1
int16(-32768)
uint64(18446744073709551615)
float64(0.333)
bool(false)
string("中文测试")
[]byte("🚀✨🧪")`,
			fuzzFunc: func(t *testing.T, a int16, b uint64, c float64, d bool, e string, f []byte) {
				if a != -32768 || b != 18446744073709551615 || c != 0.333 || d != false || e != "中文测试" || string(f) != "🚀✨🧪" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 5",
			testcase: `go test fuzz v1
int64(-1234567890)
uint8(255)
float32(42.0)
float64(1.0e10)
bool(true)
[]byte("bytes!")`,
			fuzzFunc: func(t *testing.T, a int64, b uint8, c float32, d float64, e bool, f []byte) {
				if a != -1234567890 || b != 255 || c != 42.0 || d != 1.0e10 || !e || string(f) != "bytes!" {
					t.Errorf("Mismatch: %v %v %v %v %v %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 6",
			testcase: `go test fuzz v1
int32(111)
uint16(222)
float32(3.33)
bool(false)
[]byte("✓")
string("string✓")`,
			fuzzFunc: func(t *testing.T, a int32, b uint16, c float32, d bool, e []byte, f string) {
				if a != 111 || b != 222 || c != 3.33 || d != false || string(e) != "✓" || f != "string✓" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 7",
			testcase: `go test fuzz v1
int8(10)
uint32(20)
float64(0.123456789)
bool(true)
string("emoji 😎")
[]byte("🔥💯")`,
			fuzzFunc: func(t *testing.T, a int8, b uint32, c float64, d bool, e string, f []byte) {
				if a != 10 || b != 20 || c != 0.123456789 || d != true || e != "emoji 😎" || string(f) != "🔥💯" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 8",
			testcase: `go test fuzz v1
int(1)
uint(2)
int64(3)
float64(4.4)
string("𝔘𝔫𝔦𝔠𝔬𝔡𝔢")
bool(true)`,
			fuzzFunc: func(t *testing.T, a int, b uint, c int64, d float64, e string, f bool) {
				if a != 1 || b != 2 || c != 3 || d != 4.4 || e != "𝔘𝔫𝔦𝔠𝔬𝔡𝔢" || f != true {
					t.Errorf("Mismatch: %v %v %v %v %q %v", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 9",
			testcase: `go test fuzz v1
int16(16)
uint64(64)
float32(32.32)
bool(false)
string("multi✓lingual")
[]byte("🌍📚")`,
			fuzzFunc: func(t *testing.T, a int16, b uint64, c float32, d bool, e string, f []byte) {
				if a != 16 || b != 64 || c != 32.32 || d != false || e != "multi✓lingual" || string(f) != "🌍📚" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
		{
			name: "All types 10",
			testcase: `go test fuzz v1
int64(-64)
uint8(8)
float64(99.99)
bool(true)
string("wrap-up")
[]byte("🧠🧠🧠")`,
			fuzzFunc: func(t *testing.T, a int64, b uint8, c float64, d bool, e string, f []byte) {
				if a != -64 || b != 8 || c != 99.99 || d != true || e != "wrap-up" || string(f) != "🧠🧠🧠" {
					t.Errorf("Mismatch: %v %v %v %v %q %q", a, b, c, d, e, f)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := ParseGoTestcase(tc.testcase)
			if err != nil {
				t.Fatalf("ParseGoTestcase failed: %v", err)
			}
			NewSource(data).FillAndCall(tc.fuzzFunc, reflect.ValueOf(new(testing.T)))
		})
	}
}

func TestZipCorpusFromGoFuzzCases_InlineAssert(t *testing.T) {

	t.Run("edgecase_test_10.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_10.txt"), filepath.Join(tempDir, "edgecase_test_10.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_10.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int, gotVal1 uint16, gotVal2 uint, gotVal3 int16, gotVal4 float32, gotVal5 bool) {
				if gotVal0 != 9223372036854775807 {
					t.Errorf("gotVal0 = %v; want 9223372036854775807", gotVal0)
				}
				if gotVal1 != 65535 {
					t.Errorf("gotVal1 = %v; want 65535", gotVal1)
				}
				if gotVal2 != 18446744073709551615 {
					t.Errorf("gotVal2 = %v; want 18446744073709551615", gotVal2)
				}
				if gotVal3 != -32768 {
					t.Errorf("gotVal3 = %v; want -32768", gotVal3)
				}
				if gotVal4 != 0.0 {
					t.Errorf("gotVal4 = %v; want 0.0", gotVal4)
				}
				if gotVal5 != false {
					t.Errorf("gotVal5 = %v; want False", gotVal5)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_10.txt, but it did not")
		}
	})

	t.Run("edgecase_test_11.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_11.txt"), filepath.Join(tempDir, "edgecase_test_11.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_11.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint64, gotVal1 bool, gotVal2 int, gotVal3 int16, gotVal4 uint16, gotVal5 int32, gotVal6 uint32, gotVal7 float32, gotVal8 int8, gotVal9 uint, gotVal10 int64) {
				if gotVal0 != 18446744073709551615 {
					t.Errorf("gotVal0 = %v; want 18446744073709551615", gotVal0)
				}
				if gotVal1 != false {
					t.Errorf("gotVal1 = %v; want False", gotVal1)
				}
				if gotVal2 != 9223372036854775807 {
					t.Errorf("gotVal2 = %v; want 9223372036854775807", gotVal2)
				}
				if gotVal3 != 0 {
					t.Errorf("gotVal3 = %v; want 0", gotVal3)
				}
				if gotVal4 != 65535 {
					t.Errorf("gotVal4 = %v; want 65535", gotVal4)
				}
				if gotVal5 != -2147483648 {
					t.Errorf("gotVal5 = %v; want -2147483648", gotVal5)
				}
				if gotVal6 != 4294967295 {
					t.Errorf("gotVal6 = %v; want 4294967295", gotVal6)
				}
				if gotVal7 != 0.0 {
					t.Errorf("gotVal7 = %v; want 0.0", gotVal7)
				}
				if gotVal8 != -128 {
					t.Errorf("gotVal8 = %v; want -128", gotVal8)
				}
				if gotVal9 != 18446744073709551615 {
					t.Errorf("gotVal9 = %v; want 18446744073709551615", gotVal9)
				}
				if gotVal10 != -9223372036854775808 {
					t.Errorf("gotVal10 = %v; want -9223372036854775808", gotVal10)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_11.txt, but it did not")
		}
	})

	t.Run("edgecase_test_12.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_12.txt"), filepath.Join(tempDir, "edgecase_test_12.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_12.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint8, gotVal1 uint32, gotVal2 bool, gotVal3 int, gotVal4 int8, gotVal5 float32) {
				if gotVal0 != 255 {
					t.Errorf("gotVal0 = %v; want 255", gotVal0)
				}
				if gotVal1 != 4294967295 {
					t.Errorf("gotVal1 = %v; want 4294967295", gotVal1)
				}
				if gotVal2 != false {
					t.Errorf("gotVal2 = %v; want False", gotVal2)
				}
				if gotVal3 != 9223372036854775807 {
					t.Errorf("gotVal3 = %v; want 9223372036854775807", gotVal3)
				}
				if gotVal4 != 127 {
					t.Errorf("gotVal4 = %v; want 127", gotVal4)
				}
				if gotVal5 != float32(math.Inf(-1)) {
					t.Errorf("gotVal5 = %v; want math.Inf(-1)", gotVal5)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_12.txt, but it did not")
		}
	})

	t.Run("edgecase_test_13.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_13.txt"), filepath.Join(tempDir, "edgecase_test_13.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_13.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 bool, gotVal1 uint32, gotVal2 uint8, gotVal3 float32, gotVal4 string, gotVal5 int16, gotVal6 int32, gotVal7 int8, gotVal8 uint16) {
				if gotVal0 != true {
					t.Errorf("gotVal0 = %v; want true", gotVal0)
				}
				if gotVal1 != 4294967295 {
					t.Errorf("gotVal1 = %v; want 4294967295", gotVal1)
				}
				if gotVal2 != 255 {
					t.Errorf("gotVal2 = %v; want 255", gotVal2)
				}
				if gotVal3 != float32(math.Inf(-1)) {
					t.Errorf("gotVal3 = %v; want math.Inf(-1)", gotVal3)
				}
				if gotVal4 != "bMANGgwPGeJo\u00f8bagEnGPV\u5b57YDLlMj🚀ztZ" {
					t.Errorf("gotVal4 = %q; want %q", gotVal4, "bMANGgwPGeJo\u00f8bagEnGPV\u5b57YDLlMj🚀ztZ")
				}
				if gotVal5 != -32768 {
					t.Errorf("gotVal5 = %v; want -32768", gotVal5)
				}
				if gotVal6 != 2147483647 {
					t.Errorf("gotVal6 = %v; want 2147483647", gotVal6)
				}
				if gotVal7 != -128 {
					t.Errorf("gotVal7 = %v; want -128", gotVal7)
				}
				if gotVal8 != 65535 {
					t.Errorf("gotVal8 = %v; want 65535", gotVal8)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_13.txt, but it did not")
		}
	})

	t.Run("edgecase_test_14.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_14.txt"), filepath.Join(tempDir, "edgecase_test_14.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_14.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int16, gotVal1 float64, gotVal2 []byte, gotVal3 uint16, gotVal4 int32, gotVal5 float32, gotVal6 uint8) {
				if gotVal0 != 32767 {
					t.Errorf("gotVal0 = %v; want 32767", gotVal0)
				}
				if gotVal1 != math.Inf(-1) {
					t.Errorf("gotVal1 = %v; want math.Inf(-1)", gotVal1)
				}
				if string(gotVal2) != "d\\|]8R7\fEW}h:t\u00df\t\u22069Y 📦JYMX+^I#\u00a9,b\u00df#B\f-\\zLY~{12🔬\t-{~{xc?vo🧪JRn_C1}OI\u00f1📦kB\u00dfi<G" {
					t.Errorf("gotVal2 = %q; want %q", string(gotVal2), "d\\|]8R7\fEW}h:t\u00df\t\u22069Y 📦JYMX+^I#\u00a9,b\u00df#B\f-\\zLY~{12🔬\t-{~{xc?vo🧪JRn_C1}OI\u00f1📦kB\u00dfi<G")
				}
				if gotVal3 != 65535 {
					t.Errorf("gotVal3 = %v; want 65535", gotVal3)
				}
				if gotVal4 != -2147483648 {
					t.Errorf("gotVal4 = %v; want -2147483648", gotVal4)
				}
				if gotVal5 != -1e+38 {
					t.Errorf("gotVal5 = %v; want -1e+38", gotVal5)
				}
				if gotVal6 != 255 {
					t.Errorf("gotVal6 = %v; want 255", gotVal6)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_14.txt, but it did not")
		}
	})

	t.Run("edgecase_test_15.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_15.txt"), filepath.Join(tempDir, "edgecase_test_15.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_15.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint8, gotVal1 int32, gotVal2 int16, gotVal3 float32, gotVal4 int64, gotVal5 uint32, gotVal6 int, gotVal7 uint, gotVal8 uint16, gotVal9 int8, gotVal10 bool, gotVal11 []byte) {
				if gotVal0 != 255 {
					t.Errorf("gotVal0 = %v; want 255", gotVal0)
				}
				if gotVal1 != -2147483648 {
					t.Errorf("gotVal1 = %v; want -2147483648", gotVal1)
				}
				if gotVal2 != -32768 {
					t.Errorf("gotVal2 = %v; want -32768", gotVal2)
				}
				if gotVal3 != 0.0 {
					t.Errorf("gotVal3 = %v; want 0.0", gotVal3)
				}
				if gotVal4 != 9223372036854775807 {
					t.Errorf("gotVal4 = %v; want 9223372036854775807", gotVal4)
				}
				if gotVal5 != 4294967295 {
					t.Errorf("gotVal5 = %v; want 4294967295", gotVal5)
				}
				if gotVal6 != 9223372036854775807 {
					t.Errorf("gotVal6 = %v; want 9223372036854775807", gotVal6)
				}
				if gotVal7 != 18446744073709551615 {
					t.Errorf("gotVal7 = %v; want 18446744073709551615", gotVal7)
				}
				if gotVal8 != 65535 {
					t.Errorf("gotVal8 = %v; want 65535", gotVal8)
				}
				if gotVal9 != -128 {
					t.Errorf("gotVal9 = %v; want -128", gotVal9)
				}
				if gotVal10 != false {
					t.Errorf("gotVal10 = %v; want False", gotVal10)
				}
				if string(gotVal11) != "n5K7%wt^5)Gg:sm4?P19G%Y;.=;_o\u00fc{gaq[')📦Pd_&rN\u000b0@Jm%\\a|Ez\u00df:5Ol=H" {
					t.Errorf("gotVal11 = %q; want %q", string(gotVal11), "n5K7%wt^5)Gg:sm4?P19G%Y;.=;_o\u00fc{gaq[')📦Pd_&rN\u000b0@Jm%\\a|Ez\u00df:5Ol=H")
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_15.txt, but it did not")
		}
	})

	t.Run("edgecase_test_16.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_16.txt"), filepath.Join(tempDir, "edgecase_test_16.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_16.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint8, gotVal1 float64, gotVal2 int64, gotVal3 int16, gotVal4 uint32, gotVal5 float32, gotVal6 []byte, gotVal7 int, gotVal8 uint16, gotVal9 uint) {
				if gotVal0 != 255 {
					t.Errorf("gotVal0 = %v; want 255", gotVal0)
				}
				if gotVal1 != 0.0 {
					t.Errorf("gotVal1 = %v; want 0.0", gotVal1)
				}
				if gotVal2 != -9223372036854775808 {
					t.Errorf("gotVal2 = %v; want -9223372036854775808", gotVal2)
				}
				if gotVal3 != 32767 {
					t.Errorf("gotVal3 = %v; want 32767", gotVal3)
				}
				if gotVal4 != 4294967295 {
					t.Errorf("gotVal4 = %v; want 4294967295", gotVal4)
				}
				if gotVal5 != float32(math.Inf(-1)) {
					t.Errorf("gotVal5 = %v; want math.Inf(-1)", gotVal5)
				}
				if string(gotVal6) != "-8gtw🔬\f[T|nb1O2e(f<j>\n@j}+hB|5o<@\u00df{KeY o\nI\u00dfTY#\fWME" {
					t.Errorf("gotVal6 = %q; want %q", string(gotVal6), "-8gtw🔬\f[T|nb1O2e(f<j>\n@j}+hB|5o<@\u00df{KeY o\nI\u00dfTY#\fWME")
				}
				if gotVal7 != -9223372036854775808 {
					t.Errorf("gotVal7 = %v; want -9223372036854775808", gotVal7)
				}
				if gotVal8 != 65535 {
					t.Errorf("gotVal8 = %v; want 65535", gotVal8)
				}
				if gotVal9 != 18446744073709551615 {
					t.Errorf("gotVal9 = %v; want 18446744073709551615", gotVal9)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_16.txt, but it did not")
		}
	})

	t.Run("edgecase_test_17.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_17.txt"), filepath.Join(tempDir, "edgecase_test_17.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_17.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int8, gotVal1 uint, gotVal2 float64, gotVal3 uint16, gotVal4 uint8, gotVal5 uint32) {
				if gotVal0 != 0 {
					t.Errorf("gotVal0 = %v; want 0", gotVal0)
				}
				if gotVal1 != 18446744073709551615 {
					t.Errorf("gotVal1 = %v; want 18446744073709551615", gotVal1)
				}
				if gotVal2 != math.Inf(1) {
					t.Errorf("gotVal2 = %v; want math.Inf(1)", gotVal2)
				}
				if gotVal3 != 65535 {
					t.Errorf("gotVal3 = %v; want 65535", gotVal3)
				}
				if gotVal4 != 255 {
					t.Errorf("gotVal4 = %v; want 255", gotVal4)
				}
				if gotVal5 != 4294967295 {
					t.Errorf("gotVal5 = %v; want 4294967295", gotVal5)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_17.txt, but it did not")
		}
	})

	t.Run("edgecase_test_18.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_18.txt"), filepath.Join(tempDir, "edgecase_test_18.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_18.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int8, gotVal1 int, gotVal2 int64, gotVal3 uint8, gotVal4 uint, gotVal5 uint16) {
				if gotVal0 != 127 {
					t.Errorf("gotVal0 = %v; want 127", gotVal0)
				}
				if gotVal1 != 9223372036854775807 {
					t.Errorf("gotVal1 = %v; want 9223372036854775807", gotVal1)
				}
				if gotVal2 != 9223372036854775807 {
					t.Errorf("gotVal2 = %v; want 9223372036854775807", gotVal2)
				}
				if gotVal3 != 255 {
					t.Errorf("gotVal3 = %v; want 255", gotVal3)
				}
				if gotVal4 != 18446744073709551615 {
					t.Errorf("gotVal4 = %v; want 18446744073709551615", gotVal4)
				}
				if gotVal5 != 65535 {
					t.Errorf("gotVal5 = %v; want 65535", gotVal5)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_18.txt, but it did not")
		}
	})

	t.Run("edgecase_test_19.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_19.txt"), filepath.Join(tempDir, "edgecase_test_19.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_19.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int32, gotVal1 int64, gotVal2 float64, gotVal3 int, gotVal4 int8, gotVal5 bool, gotVal6 float32, gotVal7 uint16, gotVal8 string, gotVal9 uint, gotVal10 []byte) {
				if gotVal0 != 2147483647 {
					t.Errorf("gotVal0 = %v; want 2147483647", gotVal0)
				}
				if gotVal1 != 9223372036854775807 {
					t.Errorf("gotVal1 = %v; want 9223372036854775807", gotVal1)
				}
				if gotVal2 != math.Inf(-1) {
					t.Errorf("gotVal2 = %v; want math.Inf(-1)", gotVal2)
				}
				if gotVal3 != 9223372036854775807 {
					t.Errorf("gotVal3 = %v; want 9223372036854775807", gotVal3)
				}
				if gotVal4 != 127 {
					t.Errorf("gotVal4 = %v; want 127", gotVal4)
				}
				if gotVal5 != false {
					t.Errorf("gotVal5 = %v; want False", gotVal5)
				}
				if gotVal6 != 0.0 {
					t.Errorf("gotVal6 = %v; want 0.0", gotVal6)
				}
				if gotVal7 != 65535 {
					t.Errorf("gotVal7 = %v; want 65535", gotVal7)
				}
				if gotVal8 != "RocbigN🚀🚀N\u6c49mDIpMrkjF\u00f8Shf\u00f1🚀FHSkKCSxddUzRGidwhcquoi\u5b57jGAnewclvnVl\u5b57UufWOwGaEdpOblV\u6c49l🚀QxQ\u00f8" {
					t.Errorf("gotVal8 = %q; want %q", gotVal8, "RocbigN🚀🚀N\u6c49mDIpMrkjF\u00f8Shf\u00f1🚀FHSkKCSxddUzRGidwhcquoi\u5b57jGAnewclvnVl\u5b57UufWOwGaEdpOblV\u6c49l🚀QxQ\u00f8")
				}
				if gotVal9 != 18446744073709551615 {
					t.Errorf("gotVal9 = %v; want 18446744073709551615", gotVal9)
				}
				if string(gotVal10) != "\toS}55<-/#\\LKR3I0\f\u00a9E|vUK4f1aLPUX\\RY7sf4DTS" {
					t.Errorf("gotVal10 = %q; want %q", string(gotVal10), "\toS}55<-/#\\LKR3I0\f\u00a9E|vUK4f1aLPUX\\RY7sf4DTS")
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_19.txt, but it did not")
		}
	})

	t.Run("edgecase_test_2.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_2.txt"), filepath.Join(tempDir, "edgecase_test_2.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_2.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 []byte, gotVal1 int32, gotVal2 uint, gotVal3 uint64, gotVal4 int64, gotVal5 uint8, gotVal6 string) {
				if string(gotVal0) != "LZsyUPS^a\r[bD=vPFE4lZl.:S^/\\\"I4?D[7\u000b(9\nQ<|4^c[lr\u00f1\\\\'>\\\"d)'iH'{d/\rc+75X🧪 AW^\\t[<m4O\u00dfxu55🧪m/zg\\\\\\\"St" {
					t.Errorf("gotVal0 = %q; want %q", string(gotVal0), "LZsyUPS^a\r[bD=vPFE4lZl.:S^/\\\"I4?D[7\u000b(9\nQ<|4^c[lr\u00f1\\\\'>\\\"d)'iH'{d/\rc+75X🧪 AW^\\t[<m4O\u00dfxu55🧪m/zg\\\\\\\"St")
				}
				if gotVal1 != 2147483647 {
					t.Errorf("gotVal1 = %v; want 2147483647", gotVal1)
				}
				if gotVal2 != 18446744073709551615 {
					t.Errorf("gotVal2 = %v; want 18446744073709551615", gotVal2)
				}
				if gotVal3 != 18446744073709551615 {
					t.Errorf("gotVal3 = %v; want 18446744073709551615", gotVal3)
				}
				if gotVal4 != 9223372036854775807 {
					t.Errorf("gotVal4 = %v; want 9223372036854775807", gotVal4)
				}
				if gotVal5 != 255 {
					t.Errorf("gotVal5 = %v; want 255", gotVal5)
				}
				if gotVal6 != "haNkDH\u00f1jBmNYqAZDHeUgviQGYuJje\u00f8jGDcFtZd\u00f8vfOLwiPQynZzwuhAs\u6c49xIBe\u00f8widfh🚀UTWoE\u00f1dXIToNELuZtABQvhwlcjy\u5b57IGSk" {
					t.Errorf("gotVal6 = %q; want %q", gotVal6, "haNkDH\u00f1jBmNYqAZDHeUgviQGYuJje\u00f8jGDcFtZd\u00f8vfOLwiPQynZzwuhAs\u6c49xIBe\u00f8widfh🚀UTWoE\u00f1dXIToNELuZtABQvhwlcjy\u5b57IGSk")
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_2.txt, but it did not")
		}
	})

	t.Run("edgecase_test_3.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_3.txt"), filepath.Join(tempDir, "edgecase_test_3.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_3.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 bool, gotVal1 uint32, gotVal2 uint8, gotVal3 float32, gotVal4 string, gotVal5 int16, gotVal6 int32, gotVal7 int8, gotVal8 uint16) {
				if gotVal0 != true {
					t.Errorf("gotVal0 = %v; want True", gotVal0)
				}
				if gotVal1 != 4294967295 {
					t.Errorf("gotVal1 = %v; want 4294967295", gotVal1)
				}
				if gotVal2 != 255 {
					t.Errorf("gotVal2 = %v; want 255", gotVal2)
				}
				if gotVal3 != float32(math.Inf(-1)) {
					t.Errorf("gotVal3 = %v; want math.Inf(-1)", gotVal3)
				}
				if gotVal4 != "bMANGgwPGeJo\u00f8bagEnGPV\u5b57YDLlMj🚀ztZ" {
					t.Errorf("gotVal4 = %q; want %q", gotVal4, "bMANGgwPGeJo\u00f8bagEnGPV\u5b57YDLlMj🚀ztZ")
				}
				if gotVal5 != -32768 {
					t.Errorf("gotVal5 = %v; want -32768", gotVal5)
				}
				if gotVal6 != 2147483647 {
					t.Errorf("gotVal6 = %v; want 2147483647", gotVal6)
				}
				if gotVal7 != -128 {
					t.Errorf("gotVal7 = %v; want -128", gotVal7)
				}
				if gotVal8 != 65535 {
					t.Errorf("gotVal8 = %v; want 65535", gotVal8)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_3.txt, but it did not")
		}
	})

	t.Run("edgecase_test_4.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_4.txt"), filepath.Join(tempDir, "edgecase_test_4.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_4.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint32, gotVal1 int32, gotVal2 uint, gotVal3 []byte, gotVal4 int8, gotVal5 uint8, gotVal6 int64, gotVal7 int16, gotVal8 float64) {
				if gotVal0 != 4294967295 {
					t.Errorf("gotVal0 = %v; want 4294967295", gotVal0)
				}
				if gotVal1 != -2147483648 {
					t.Errorf("gotVal1 = %v; want -2147483648", gotVal1)
				}
				if gotVal2 != 18446744073709551615 {
					t.Errorf("gotVal2 = %v; want 18446744073709551615", gotVal2)
				}
				if string(gotVal3) != "/,5\u00f1nP🧪jy+A=9?fpD=:WSrx|🧪s*`Q\u00a9ia~>\u00a9\\1o=6QN4S@" {
					t.Errorf("gotVal3 = %q; want %q", string(gotVal3), "/,5\u00f1nP🧪jy+A=9?fpD=:WSrx|🧪s*`Q\u00a9ia~>\u00a9\\1o=6QN4S@")
				}
				if gotVal4 != -128 {
					t.Errorf("gotVal4 = %v; want -128", gotVal4)
				}
				if gotVal5 != 255 {
					t.Errorf("gotVal5 = %v; want 255", gotVal5)
				}
				if gotVal6 != 9223372036854775807 {
					t.Errorf("gotVal6 = %v; want 9223372036854775807", gotVal6)
				}
				if gotVal7 != 0 {
					t.Errorf("gotVal7 = %v; want 0", gotVal7)
				}
				if gotVal8 != 0.0 {
					t.Errorf("gotVal8 = %v; want 0.0", gotVal8)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_4.txt, but it did not")
		}
	})

	t.Run("edgecase_test_5.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_5.txt"), filepath.Join(tempDir, "edgecase_test_5.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_5.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint16, gotVal1 bool, gotVal2 uint8, gotVal3 int8, gotVal4 int, gotVal5 float32, gotVal6 float64) {
				if gotVal0 != 65535 {
					t.Errorf("gotVal0 = %v; want 65535", gotVal0)
				}
				if gotVal1 != true {
					t.Errorf("gotVal1 = %v; want True", gotVal1)
				}
				if gotVal2 != 255 {
					t.Errorf("gotVal2 = %v; want 255", gotVal2)
				}
				if gotVal3 != -128 {
					t.Errorf("gotVal3 = %v; want -128", gotVal3)
				}
				if gotVal4 != 9223372036854775807 {
					t.Errorf("gotVal4 = %v; want 9223372036854775807", gotVal4)
				}
				if gotVal5 != float32(math.Inf(1)) {
					t.Errorf("gotVal5 = %v; want math.Inf(1)", gotVal5)
				}
				if gotVal6 != math.Inf(-1) {
					t.Errorf("gotVal6 = %v; want math.Inf(-1)", gotVal6)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_5.txt, but it did not")
		}
	})

	t.Run("edgecase_test_6.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_6.txt"), filepath.Join(tempDir, "edgecase_test_6.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_6.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int8, gotVal1 int, gotVal2 uint, gotVal3 uint32, gotVal4 uint64, gotVal5 string, gotVal6 float64, gotVal7 int64, gotVal8 int32, gotVal9 []byte) {
				if gotVal0 != -128 {
					t.Errorf("gotVal0 = %v; want -128", gotVal0)
				}
				if gotVal1 != 9223372036854775807 {
					t.Errorf("gotVal1 = %v; want 9223372036854775807", gotVal1)
				}
				if gotVal2 != 18446744073709551615 {
					t.Errorf("gotVal2 = %v; want 18446744073709551615", gotVal2)
				}
				if gotVal3 != 4294967295 {
					t.Errorf("gotVal3 = %v; want 4294967295", gotVal3)
				}
				if gotVal4 != 18446744073709551615 {
					t.Errorf("gotVal4 = %v; want 18446744073709551615", gotVal4)
				}
				if gotVal5 != "sQ\u00f1aojKLPcp\u00f8hrM\u00f1pDzX" {
					t.Errorf("gotVal5 = %q; want %q", gotVal5, "sQ\u00f1aojKLPcp\u00f8hrM\u00f1pDzX")
				}
				if gotVal6 != math.Inf(-1) {
					t.Errorf("gotVal6 = %v; want math.Inf(-1)", gotVal6)
				}
				if gotVal7 != -9223372036854775808 {
					t.Errorf("gotVal7 = %v; want -9223372036854775808", gotVal7)
				}
				if gotVal8 != -2147483648 {
					t.Errorf("gotVal8 = %v; want -2147483648", gotVal8)
				}
				if string(gotVal9) != "\"$d" {
					t.Errorf("gotVal9 = %q; want %q", string(gotVal9), "\"$d")
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_6.txt, but it did not")
		}
	})

	t.Run("edgecase_test_7.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_7.txt"), filepath.Join(tempDir, "edgecase_test_7.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_7.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 int8, gotVal1 int64, gotVal2 uint16, gotVal3 uint32, gotVal4 float32, gotVal5 uint64, gotVal6 uint, gotVal7 bool, gotVal8 int32) {
				if gotVal0 != -128 {
					t.Errorf("gotVal0 = %v; want -128", gotVal0)
				}
				if gotVal1 != 9223372036854775807 {
					t.Errorf("gotVal1 = %v; want 9223372036854775807", gotVal1)
				}
				if gotVal2 != 65535 {
					t.Errorf("gotVal2 = %v; want 65535", gotVal2)
				}
				if gotVal3 != 4294967295 {
					t.Errorf("gotVal3 = %v; want 4294967295", gotVal3)
				}
				if gotVal4 != 0.0 {
					t.Errorf("gotVal4 = %v; want 0.0", gotVal4)
				}
				if gotVal5 != 18446744073709551615 {
					t.Errorf("gotVal5 = %v; want 18446744073709551615", gotVal5)
				}
				if gotVal6 != 18446744073709551615 {
					t.Errorf("gotVal6 = %v; want 18446744073709551615", gotVal6)
				}
				if gotVal7 != true {
					t.Errorf("gotVal7 = %v; want True", gotVal7)
				}
				if gotVal8 != 2147483647 {
					t.Errorf("gotVal8 = %v; want 2147483647", gotVal8)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_7.txt, but it did not")
		}
	})

	t.Run("edgecase_test_8.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_8.txt"), filepath.Join(tempDir, "edgecase_test_8.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_8.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 float32, gotVal1 int64, gotVal2 int16, gotVal3 string, gotVal4 uint32, gotVal5 uint64) {
				if gotVal0 != float32(math.Inf(1)) {
					t.Errorf("gotVal0 = %v; want math.Inf(1)", gotVal0)
				}
				if gotVal1 != 9223372036854775807 {
					t.Errorf("gotVal1 = %v; want 9223372036854775807", gotVal1)
				}
				if gotVal2 != 32767 {
					t.Errorf("gotVal2 = %v; want 32767", gotVal2)
				}
				if gotVal3 != "sSTJeh\u5b57btG\u6c49RfDadVGsHOvCDIQJqnGssicYcyGbGvFgwnUrr\u00f1fxxSSLi\u5b57sPYyuPwzygRiTlkguwoTZVfD\u00f1EEwdGPjPr" {
					t.Errorf("gotVal3 = %q; want %q", gotVal3, "sSTJeh\u5b57btG\u6c49RfDadVGsHOvCDIQJqnGssicYcyGbGvFgwnUrr\u00f1fxxSSLi\u5b57sPYyuPwzygRiTlkguwoTZVfD\u00f1EEwdGPjPr")
				}
				if gotVal4 != 4294967295 {
					t.Errorf("gotVal4 = %v; want 4294967295", gotVal4)
				}
				if gotVal5 != 18446744073709551615 {
					t.Errorf("gotVal5 = %v; want 18446744073709551615", gotVal5)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_8.txt, but it did not")
		}
	})

	t.Run("edgecase_test_9.txt", func(t *testing.T) {
		tempDir := t.TempDir()
		outDir := t.TempDir()
		t.Setenv("OUT", outDir)

		copyFile(t, filepath.Join("testdata", "edgecase_test_9.txt"), filepath.Join(tempDir, "edgecase_test_9.txt"))

		if err := ZipCorpusFromGoFuzzCases(tempDir, "zipfuzz", false); err != nil {
			t.Fatalf("failed to create zip: %v", err)
		}

		zipReader, err := zip.OpenReader(filepath.Join(outDir, "zipfuzz.zip"))
		if err != nil {
			t.Fatalf("failed to open zip: %v", err)
		}
		defer zipReader.Close()

		found := false
		for _, file := range zipReader.File {
			if file.Name != "edgecase_test_9.txt" {
				continue
			}
			found = true
			r, err := file.Open()
			if err != nil {
				t.Fatalf("failed to open zip entry: %v", err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatalf("failed to read zip entry: %v", err)
			}

			fuzzFunc := func(t *testing.T, gotVal0 uint16, gotVal1 int16, gotVal2 int, gotVal3 int8, gotVal4 float32, gotVal5 []byte, gotVal6 int64, gotVal7 uint, gotVal8 int32) {
				if gotVal0 != 65535 {
					t.Errorf("gotVal0 = %v; want 65535", gotVal0)
				}
				if gotVal1 != -32768 {
					t.Errorf("gotVal1 = %v; want -32768", gotVal1)
				}
				if gotVal2 != 9223372036854775807 {
					t.Errorf("gotVal2 = %v; want 9223372036854775807", gotVal2)
				}
				if gotVal3 != -128 {
					t.Errorf("gotVal3 = %v; want -128", gotVal3)
				}
				if gotVal4 != 0.0 {
					t.Errorf("gotVal4 = %v; want 0.0", gotVal4)
				}
				if string(gotVal5) != "Gd-iy+Tb🔬(3" {
					t.Errorf("gotVal5 = %q; want %q", string(gotVal5), "Gd-iy+Tb🔬(3")
				}
				if gotVal6 != 9223372036854775807 {
					t.Errorf("gotVal6 = %v; want 9223372036854775807", gotVal6)
				}
				if gotVal7 != 18446744073709551615 {
					t.Errorf("gotVal7 = %v; want 18446744073709551615", gotVal7)
				}
				if gotVal8 != 2147483647 {
					t.Errorf("gotVal8 = %v; want 2147483647", gotVal8)
				}
			}
			NewSource(data).FillAndCall(fuzzFunc, reflect.ValueOf(new(testing.T)))
			break
		}
		if !found {
			t.Fatalf("expected zip to contain edgecase_test_9.txt, but it did not")
		}
	})
}

func copyFile(t *testing.T, src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("failed to open source file: %v", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("failed to create destination file: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}
}
