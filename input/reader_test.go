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

func TestCorpusConversion(t *testing.T) {
	var have string

	for _, tc := range []struct {
		want string
		data string
		wantTestcase string
		fuzzFunc func(t *testing.T, a, b string, c []byte, d, e int, f uint32, g uint64)
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
		name      string
		testcase  string
		fuzzFunc  any
		expected  func(t *testing.T)
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