package testing

type TB struct{}

func (c *B) Cleanup(func())                    {}
func (c *B) Error(args ...any)                 {}
func (c *B) Errorf(format string, args ...any) {}
func (c *B) Fail()                             {}
func (c *B) FailNow()                          {}
func (c *B) Failed() bool                      {}
func (c *B) Fatal(args ...any)                 {}
func (c *B) Fatalf(format string, args ...any) {}
func (c *B) Helper()                           {}
func (c *B) Log(args ...any)                   {}
func (c *B) Logf(format string, args ...any)   {}
func (c *B) Name() string                      { return "Fuzz" }
func (c *B) Setenv(key, value string)          {}
func (c *B) Skip(args ...any)                  {}
func (c *B) SkipNow()                          {}
func (c *B) Skipf(format string, args ...any)  {}
func (c *B) Skipped() bool                     {}
func (c *B) TempDir() string                   {}
