package utils

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
)

type T struct {
	parent   *T
	name     string
	mu       sync.RWMutex
	skipped  bool
	failed   bool
	finished bool
	cleanups []func()
}

// Most of the T functions are copied from the stdlib

var errNilPanicOrGoexit = errors.New("test executed panic(nil) or runtime.Goexit")

type panicHandling int

const (
	normalPanic panicHandling = iota
	recoverAndReturnPanic
)

func (t *T) log(s string) {}
func (t *T) Cleanup(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cleanups = append(t.cleanups, f)
}
func (t *T) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }
func (t *T) Log(args ...any) {
	t.log(fmt.Sprintln(args...))
}
func (t *T) Logf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
}
func (t *T) Error(args ...any) {
	t.log(fmt.Sprintln(args...))
	t.Fail()
}
func (t *T) Errorf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
	t.Fail()
}
func (t *T) Fatal(args ...any) {
	t.log(fmt.Sprintln(args...))
	t.FailNow()
}
func (t *T) Fatalf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
	t.FailNow()
}
func (t *T) Skip(args ...any) {
	t.log(fmt.Sprintln(args...))
	t.SkipNow()
}
func (t *T) Skipf(format string, args ...any) {
	t.log(fmt.Sprintf(format, args...))
	t.SkipNow()
}
func (t *T) Fail() {
	if t.parent != nil {
		t.parent.Fail()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed = true
}
func (t *T) FailNow() {
	t.Fail()
	t.mu.Lock()
	t.finished = true
	t.mu.Unlock()
	runtime.Goexit()
}
func (t *T) Failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}
func (t *T) Helper()      {}
func (t *T) Name() string { return t.name }
func (t *T) Parallel()    {}
func (t *T) Run(name string, f func(t *T)) bool {
	s := new(T)
	s.parent = t
	s.name = t.name + "/" + name
	done := make(chan struct{})
	go s.run(done, f)
	<-done
	return !s.failed
}

func (t *T) run(done chan<- struct{}, f func(t *T)) {
	defer close(done)
	defer func() {
		err := recover()

		t.mu.RLock()
		finished := t.finished
		t.mu.RUnlock()
		if !finished && err == nil {
			err = errNilPanicOrGoexit
			for p := t.parent; p != nil; p = p.parent {
				p.mu.RLock()
				finished = p.finished
				p.mu.RUnlock()
				if finished {
					t.Errorf("%v: subtest may have called FailNow on a parent test", err)
					err = nil
					break
				}
			}
		}

		if err == nil {
			return
		}

		prefix := "panic: "
		if err == errNilPanicOrGoexit {
			prefix = ""
		}
		t.Errorf("%s%s\n%s\n", prefix, err, string(debug.Stack()))
		t.mu.Lock()
		t.finished = true
		t.mu.Unlock()
	}()
	defer t.runCleanup(normalPanic)

	f(t)

	t.mu.Lock()
	t.finished = true
	t.mu.Unlock()
}
func (t *T) runCleanup(ph panicHandling) (panicVal any) {
	if ph == recoverAndReturnPanic {
		defer func() {
			panicVal = recover()
		}()
	}

	// Make sure that if a cleanup function panics,
	// we still run the remaining cleanup functions.
	defer func() {
		t.mu.Lock()
		recur := len(t.cleanups) > 0
		t.mu.Unlock()
		if recur {
			t.runCleanup(normalPanic)
		}
	}()

	for {
		var cleanup func()
		t.mu.Lock()
		if len(t.cleanups) > 0 {
			last := len(t.cleanups) - 1
			cleanup = t.cleanups[last]
			t.cleanups = t.cleanups[:last]
		}
		t.mu.Unlock()
		if cleanup == nil {
			return nil
		}
		cleanup()
	}
}
func (t *T) Setenv(key, value string) {}
func (t *T) markSkipped() {
	t.mu.Lock()
	t.skipped = true
	t.finished = true
	t.mu.Unlock()
}
func (t *T) SkipNow() {
	t.markSkipped()
	runtime.Goexit()
}
func (t *T) Skipped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.skipped
}
func (t *T) TempDir() string { return "/tmp" }

type F struct {
	T
	data    []byte
	capture bool
	values  []any
	corpus  [][]any
}

func NewF(name string, data []byte) *F {
	return &F{T: T{name: name}, data: data}
}

func NewCaptureF(name string, data []byte) *F {
	return &F{T: T{name: name}, data: data, capture: true}
}

func (f *F) Add(args ...any) { f.corpus = append(f.corpus, args) }
func (f *F) Fuzz(ff any) {
	// we are assuming that ff is a func.
	// TODO: Add a check for UX purposes

	fn := reflect.ValueOf(ff)
	fnType := fn.Type()
	var types []reflect.Type
	for i := 1; i < fnType.NumIn(); i++ {
		t := fnType.In(i)

		types = append(types, t)
	}
	args := []reflect.Value{reflect.ValueOf(&f.T)}
	fuzzConsumer := fuzz.NewConsumer(f.data)
	for _, v := range types {
		switch v.String() {
		case "[]uint8":
			b, err := fuzzConsumer.GetBytes()
			if err != nil {
				f.markSkipped()
				return
			}
			newBytes := reflect.New(v)
			newBytes.Elem().SetBytes(b)
			args = append(args, newBytes.Elem())
			f.values = append(f.values, newBytes.Elem().Interface())
		case "string":
			s, err := fuzzConsumer.GetString()
			if err != nil {
				f.markSkipped()
				return
			}
			newString := reflect.New(v)
			newString.Elem().SetString(s)
			args = append(args, newString.Elem())
			f.values = append(f.values, newString.Elem().Interface())
		case "int":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int64(randInt))
			args = append(args, newInt.Elem())
			f.values = append(f.values, newInt.Elem().Interface())
		case "int8":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int64(randInt))
			args = append(args, newInt.Elem())
			f.values = append(f.values, newInt.Elem().Interface())
		case "int16":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int64(randInt))
			args = append(args, newInt.Elem())
			f.values = append(f.values, newInt.Elem().Interface())
		case "int32":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int64(randInt))
			args = append(args, newInt.Elem())
			f.values = append(f.values, newInt.Elem().Interface())
		case "int64":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newInt := reflect.New(v)
			newInt.Elem().SetInt(int64(randInt))
			args = append(args, newInt.Elem())
			f.values = append(f.values, newInt.Elem().Interface())
		case "uint":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newUint := reflect.New(v)
			newUint.Elem().SetUint(uint64(randInt))
			args = append(args, newUint.Elem())
			f.values = append(f.values, newUint.Elem().Interface())
		case "uint8":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				f.markSkipped()
				return
			}
			newUint := reflect.New(v)
			newUint.Elem().SetUint(uint64(randInt))
			args = append(args, newUint.Elem())
			f.values = append(f.values, newUint.Elem().Interface())
		case "uint16":
			randInt, err := fuzzConsumer.GetUint16()
			if err != nil {
				f.markSkipped()
				return
			}
			newUint16 := reflect.New(v)
			newUint16.Elem().SetUint(uint64(randInt))
			args = append(args, newUint16.Elem())
			f.values = append(f.values, newUint16.Elem().Interface())
		case "uint32":
			randInt, err := fuzzConsumer.GetUint32()
			if err != nil {
				f.markSkipped()
				return
			}
			newUint32 := reflect.New(v)
			newUint32.Elem().SetUint(uint64(randInt))
			args = append(args, newUint32.Elem())
			f.values = append(f.values, newUint32.Elem().Interface())
		case "uint64":
			randInt, err := fuzzConsumer.GetUint64()
			if err != nil {
				f.markSkipped()
				return
			}
			newUint64 := reflect.New(v)
			newUint64.Elem().SetUint(uint64(randInt))
			args = append(args, newUint64.Elem())
			f.values = append(f.values, newUint64.Elem().Interface())
		case "rune":
			randRune, err := fuzzConsumer.GetRune()
			if err != nil {
				f.markSkipped()
				return
			}
			newRune := reflect.New(v)
			newRune.Elem().Set(reflect.ValueOf(randRune))
			args = append(args, newRune.Elem())
			f.values = append(f.values, newRune.Elem().Interface())
		case "float32":
			randFloat, err := fuzzConsumer.GetFloat32()
			if err != nil {
				f.markSkipped()
				return
			}
			newFloat := reflect.New(v)
			newFloat.Elem().Set(reflect.ValueOf(randFloat))
			args = append(args, newFloat.Elem())
			f.values = append(f.values, newFloat.Elem().Interface())
		case "float64":
			randFloat, err := fuzzConsumer.GetFloat64()
			if err != nil {
				f.markSkipped()
				return
			}
			newFloat := reflect.New(v)
			newFloat.Elem().Set(reflect.ValueOf(randFloat))
			args = append(args, newFloat.Elem())
			f.values = append(f.values, newFloat.Elem().Interface())
		case "bool":
			randBool, err := fuzzConsumer.GetBool()
			if err != nil {
				f.markSkipped()
				return
			}
			newBool := reflect.New(v)
			newBool.Elem().Set(reflect.ValueOf(randBool))
			args = append(args, newBool.Elem())
			f.values = append(f.values, newBool.Elem().Interface())
		default:
			fmt.Println(v.String())
		}
	}

	if f.capture {
		return
	}

	done := make(chan struct{})
	go f.run(done, func(*T) { fn.Call(args) })
	<-done
}
