package testing

import (
	"time"
)

type B struct{}

func (c *B) Cleanup(f func())                    {}
func (b *B) Elapsed() time.Duration              { return time.Since(time.Now()) }
func (c *B) Error(args ...any)                   {}
func (c *B) Errorf(format string, args ...any)   {}
func (c *B) Fail()                               {}
func (c *B) FailNow()                            {}
func (c *B) Failed() bool                        { return true }
func (c *B) Fatal(args ...any)                   {}
func (c *B) Fatalf(format string, args ...any)   {}
func (c *B) Helper()                             {}
func (c *B) Log(args ...any)                     {}
func (c *B) Logf(format string, args ...any)     {}
func (c *B) Name() string                        { return "HI" }
func (b *B) ReportAllocs()                       {}
func (b *B) ReportMetric(n float64, unit string) {}
func (b *B) ResetTimer()                         {}
func (b *B) Run(name string, f func(b *B)) bool  { return true }
func (b *B) RunParallel(body func(*PB))          {}
func (b *B) SetBytes(n int64)                    {}
func (b *B) SetParallelism(p int)                {}
func (c *B) Setenv(key, value string)            {}
func (c *B) Skip(args ...any)                    {}
func (c *B) SkipNow()                            {}
func (c *B) Skipf(format string, args ...any)    {}
func (c *B) Skipped() bool                       { return true }
func (b *B) StartTimer()                         {}
func (b *B) StopTimer()                          {}
func (c *B) TempDir() string                     { return "NONE" }
