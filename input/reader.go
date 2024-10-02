package input

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
)

// Source takes a byteslice, and arguments can be pulled from it.
type Source struct {
	s         []byte
	i         int64 // current reading index
	exhausted bool
}

func NewSource(data []byte) *Source {
	return &Source{data, 0, false}
}

// IsExhausted returns true if we tried to read more data than this source
// could deliver.
func (s *Source) IsExhausted() bool {
	return s.exhausted
}

// Len returns the number of bytes of the unread portion of the data.
func (s *Source) Len() int {
	if s.i >= int64(len(s.s)) {
		return 0
	}
	return int(int64(len(s.s)) - s.i)
}

// Used returns the number of bytes already consumed.
func (s *Source) Used() int {
	return int(s.i)
}

// Read implements the io.Reader interface.
func (s *Source) Read(b []byte) (n int, err error) {
	if s.i >= int64(len(s.s)) {
		n, err = 0, io.EOF
	} else {
		n = copy(b, s.s[s.i:])
		s.i += int64(n)
	}
	if n < len(b) {
		s.exhausted = true
	}
	return n, err
}

// getBytes returns a slice of size bytes, as a direct reference if possible.
func (s *Source) getBytes(size int) []byte {
	if end := int(s.i) + size; end < len(s.s) { // Fast-path, no-copy deliver
		pos := s.i
		s.i += int64(size)
		return s.s[pos:end]
	}
	// Slow path
	buf := make([]byte, size)
	s.Read(buf)
	return buf
}

// readInt reads a signed integer from the source
func (s *Source) readInt(num reflect.Kind) int64 {
	switch num {
	case reflect.Int8:
		return int64(int8(s.getBytes(1)[0]))
	case reflect.Int16:
		return int64(int16(binary.BigEndian.Uint16(s.getBytes(2))))
	case reflect.Int32:
		return int64(int32(binary.BigEndian.Uint32(s.getBytes(4))))
	case reflect.Int64, reflect.Int:
		return int64(binary.BigEndian.Uint64(s.getBytes(8)))
	}
	panic(fmt.Sprintf("unsupported type: %v", num))
}

// readUint reads an unsigned integer from the source
func (s *Source) readUint(num reflect.Kind) uint64 {
	switch num {
	case reflect.Uint8:
		return uint64(uint8(s.getBytes(1)[0]))
	case reflect.Uint16:
		return uint64(binary.BigEndian.Uint16(s.getBytes(2)))
	case reflect.Uint32:
		return uint64(binary.BigEndian.Uint32(s.getBytes(4)))
	case reflect.Uint, reflect.Uint64:
		return binary.BigEndian.Uint64(s.getBytes(8))
	}
	panic(fmt.Sprintf("unsupported type: %v", num))
}

// FillAndCall fills the argument for the given ff (which is supposed to be a function),
// and then invokes the function.
// It returns 'true' if the function was invoked. A return-value of false means
// that the method was not invoked: probably because of insufficient input.
func (s *Source) FillAndCall(ff any, arg0 reflect.Value) (ok bool) {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	args := make([]reflect.Value, method.NumIn())
	args[0] = arg0
	var dynamic []int
	// Fill all fixed-size arguments first, then dynamic-sized fields.
	for i := 1; i < method.NumIn(); i++ {
		v := method.In(i)
		if v.Kind() <= reflect.Float64 { // fixed-size
			args[i] = s.fillArg(v, 0)
		} else { // dynamic or panic later
			dynamic = append(dynamic, i)
		}
	}
	// Second loop to fill dynamic-sized stuff
	// For filling the dynamic fields.
	// If we have only one field, it should get all the remaining input.
	// If we have N, then,
	// 1. Read N bytes [b1, b2, b3 .. bn] .
	// 2. Let the relative weights of b determine how much of the
	//    remaining input that field n gets
	weights := s.getBytes(len(dynamic))
	sum := 0
	for _, v := range weights {
		sum += int(v)
	}
	bytesLeft := s.Len()
	for i, argNum := range dynamic {
		if i == len(dynamic)-1 { // last element, it get's all that if left
			args[argNum] = s.fillArg(method.In(argNum), s.Len())
			break
		}
		var argSize = bytesLeft / len(dynamic)
		if sum > 0 {
			argSize = (bytesLeft * int(weights[i])) / sum
		}
		args[argNum] = s.fillArg(method.In(argNum), argSize)
	}
	fn.Call(args)
	return true
}

func (s *Source) fillArg(v reflect.Type, max int) reflect.Value {
	newElem := reflect.New(v).Elem()
	switch k := v.Kind(); k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		newElem.SetInt(s.readInt(k))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		newElem.SetUint(s.readUint(k))
	case reflect.Float32:
		newElem.Set(reflect.ValueOf(math.Float32frombits(uint32(s.readUint(reflect.Uint32)))))
	case reflect.Float64:
		newElem.Set(reflect.ValueOf(math.Float64frombits(s.readUint(reflect.Uint64))))
	case reflect.Bool:
		newElem.Set(reflect.ValueOf(s.readUint(reflect.Uint8)&0x1 != 0))
	case reflect.String:
		newElem.SetString(string(s.getBytes(max)))
	case reflect.Slice:
		if v.Elem().Kind() == reflect.Uint8 { // []byte
			newElem.SetBytes(s.getBytes(max))
		} else {
			panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
	}
	return newElem
}