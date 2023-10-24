package testing

import (
	"time"
)

type B struct{}

func (b *B) Cleanup(f func())                  {}
func (b *B) Error(args ...any)                 {}
func (b *B) Errorf(format string, args ...any) {}
func (b *B) Fail()                             {}
func (b *B) FailNow()                          {}
func (b *B) Failed() bool                      { panic("not implemented") }
func (b *B) Fatal(args ...any)                 {}
func (b *B) Fatalf(format string, args ...any) {}
func (b *B) Helper()                           {}
func (b *B) Log(args ...any)                   {}
func (b *B) Logf(format string, args ...any)   {}
func (b *B) Name() string                      { panic("not implemented") }
func (b *B) Setenv(key, value string)          {}
func (b *B) Skip(args ...any)                  {}
func (b *B) SkipNow()                          {}
func (b *B) Skipf(format string, args ...any)  {}
func (b *B) Skipped() bool                     { panic("not implemented") }
func (b *B) TempDir() string                   { panic("not implemented") }

func (b *B) StartTimer()                         {}
func (b *B) StopTimer()                          {}
func (b *B) ResetTimer()                         {}
func (b *B) SetBytes(n int64)                    {}
func (b *B) ReportAllocs()                       {}
func (b *B) Elapsed() time.Duration              { return 0 }
func (b *B) ReportMetric(n float64, unit string) {}
func (b *B) Run(name string, f func(b *B)) bool  { panic("not implemented") }
func (b *B) SetParallelism(p int)                {}
