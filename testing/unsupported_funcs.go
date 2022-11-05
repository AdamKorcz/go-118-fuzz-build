package testing

func AllocsPerRun(runs int, f func()) (avg float64) {
	panic(unsupportedApi("testing.AllocsPerRun"))
}
func CoverMode() string {
	panic(unsupportedApi("testing.CoverMode"))
}
func Coverage() float64 {
	panic(unsupportedApi("testing.Coverage"))	
}
func Init() {
	panic(unsupportedApi("testing.Init"))

}
func Main(matchString func(pat, str string) (bool, error), tests []InternalTest, ...) {
	panic(unsupportedApi("testing.Main"))
}
func RegisterCover(c Cover) {
	panic(unsupportedApi("testing.RegisterCover"))
}
func RunBenchmarks(matchString func(pat, str string) (bool, error), ...) {
	panic(unsupportedApi("testing.RunBenchmarks"))
}

func RunExamples(matchString func(pat, str string) (bool, error), examples []InternalExample) (ok bool) {
	panic(unsupportedApi("testing.RunExamples"))
}

func RunTests(matchString func(pat, str string) (bool, error), tests []InternalTest) (ok bool) {
	panic(unsupportedApi("testing.RunTests"))
}

func Short() bool {
	panic(unsupportedApi("testing.Short"))
}

func Verbose() bool {
	panic(unsupportedApi("testing.Verbose"))
}

type M struct {}
func (m *M) Run() (code int) {
	panic("testing.M is not support in libFuzzer Mode")
}
type PB
func (pb *PB) Next() bool {
	panic("testing.PB is not supported in libFuzzer Mode")
}