package testing

import (
	"time"
)

type B struct{}

func (b *B) StartTimer()                         {}
func (b *B) StopTimer()                          {}
func (b *B) ResetTimer()                         {}
func (b *B) SetBytes(n int64)                    {}
func (b *B) ReportAllocs()                       {}
func (b *B) Elapsed() time.Duration              { return 0 }
func (b *B) ReportMetric(n float64, unit string) {}
func (b *B) Run(name string, f func(b *B)) bool  { panic("not implemented") }
func (b *B) SetParallelism(p int)                {}
