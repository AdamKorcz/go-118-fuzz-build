package testing

type TB struct{}

func (c *TB) Cleanup(func())                    {}
func (c *TB) Error(args ...any)                 {}
func (c *TB) Errorf(format string, args ...any) {}
func (c *TB) Fail()                             {}
func (c *TB) FailNow()                          {}
func (c *TB) Failed() bool                      {}
func (c *TB) Fatal(args ...any)                 {}
func (c *TB) Fatalf(format string, args ...any) {}
func (c *TB) Helper()                           {}
func (c *TB) Log(args ...any)                   {}
func (c *TB) Logf(format string, args ...any)   {}
func (c *TB) Name() string                      { return "Fuzz" }
func (c *TB) Setenv(key, value string)          {}
func (c *TB) Skip(args ...any)                  {}
func (c *TB) SkipNow()                          {}
func (c *TB) Skipf(format string, args ...any)  {}
func (c *TB) Skipped() bool                     {}
func (c *TB) TempDir() string                   {}
