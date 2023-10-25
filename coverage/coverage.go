package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
)

type Walker struct {
	args       []string
	fuzzerName string
	fset       *token.FileSet
	src        []byte // file contents
}

// Main walker func to traverse a fuzz harness when obtaining
// the fuzzers args. Does not add the first add (t *testing.T)
func (walker *Walker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return walker
	}
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name.Name == walker.fuzzerName {
			bw := &BodyWalker{
				args:       make([]string, 0),
				fuzzerName: walker.fuzzerName,
				fset:       walker.fset,
				src:        walker.src,
			}
			ast.Walk(bw, n.Body)
			walker.args = bw.args
		}
	}
	return walker
}

type BodyWalker struct {
	args       []string
	fuzzerName string
	fset       *token.FileSet
	src        []byte // file contents
}

func (walker *BodyWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return walker
	}
	switch n := node.(type) {
	case *ast.CallExpr:
		if aa, ok := n.Fun.(*ast.SelectorExpr); ok {
			if _, ok := aa.X.(*ast.Ident); ok {
				if aa.X.(*ast.Ident).Name == "f" && aa.Sel.Name == "Fuzz" {

					// Get the func() arg to f.Fuzz:
					funcArg := n.Args[0].(*ast.FuncLit)

					walker.addArgs(funcArg.Type.Params.List[1:])
				}
			}
		}
	}
	return walker
}

// Receives a list of *ast.Field and adds them to the walker
func (walker *BodyWalker) addArgs(n []*ast.Field) {
	for _, names := range n {
		for _, _ = range names.Names {
			if a, ok := names.Type.(*ast.ArrayType); ok {
				walker.addArg(getArrayType(a))
			} else {
				walker.addArg(names.Type.(*ast.Ident).Name)
			}
		}
	}
}

func (walker *BodyWalker) addArg(arg string) {
	walker.args = append(walker.args, arg)
}

func getArrayType(n *ast.ArrayType) string {
	typeName := n.Elt.(*ast.Ident).Name
	return fmt.Sprintf("[]%s", typeName)
}

func getFuzzArgs(fuzzerFileContents, fuzzerName string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fuzz_test.go", fuzzerFileContents, 0)
	if err != nil {
		panic(err)
	}
	w := &Walker{
		args:       []string{},
		fuzzerName: fuzzerName,
		fset:       fset,
		src:        []byte(fuzzerFileContents),
	}
	ast.Walk(w, f)
	return w.args, nil
}

// This is the API that should be called externally.
// Params:
// fuzzerFileContents: the contents of the fuzzerfile. This should be
// obtained with os.ReadFile().
// testCase: The libFuzzer testcase. This should also be obtained
// with os.ReadFile().
func ConvertLibfuzzerSeedToGoSeed(fuzzerFileContents, testCase []byte, fuzzerName string) string {
	args, err := getFuzzArgs(string(fuzzerFileContents), fuzzerName)
	if err != nil {
		panic(err)
	}
	newSeed := libFuzzerSeedToGoSeed(testCase, args)
	return newSeed
}

// Takes a libFuzzer testcase and returns a Native Go testcase
func libFuzzerSeedToGoSeed(testcase []byte, args []string) string {
	var b strings.Builder
	b.WriteString("go test fuzz v1\n")

	fuzzConsumer := fuzz.NewConsumer(testcase)
	for argNumber, arg := range args {
		//fmt.Println(argNumber)
		switch arg {
		case "[]uint8", "[]byte":
			randBytes, err := fuzzConsumer.GetBytes()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "[]byte(%q)", string(randBytes))
		case "string":
			s, err := fuzzConsumer.GetString()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "string(%q)", string(s))
		case "int":
			randInt, err := fuzzConsumer.GetUint64()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "int(%v)", int(randInt))
		case "int8":
			randInt, err := fuzzConsumer.GetByte()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "int8(%v)", int8(randInt))
		case "int16":
			randInt, err := fuzzConsumer.GetUint16()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "int16(%v)", int16(randInt))
		case "int32":
			randInt, err := fuzzConsumer.GetUint32()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "int32(%v)", int32(randInt))
		case "int64":
			randInt, err := fuzzConsumer.GetUint64()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "int64(%v)", int64(randInt))
		case "uint":
			randInt, err := fuzzConsumer.GetUint64()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "uint(%v)", uint(randInt))
		case "uint8":
			randInt, err := fuzzConsumer.GetInt()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "uint8(%v)", uint8(randInt))
		case "uint16":
			randInt, err := fuzzConsumer.GetUint16()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "uint16(%v)", uint16(randInt))
		case "uint32":
			randInt, err := fuzzConsumer.GetUint32()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "uint32(%v)", uint32(randInt))
		case "uint64":
			randInt, err := fuzzConsumer.GetUint64()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "uint64(%v)", uint64(randInt))
		case "rune":
			randRune, err := fuzzConsumer.GetRune()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "rune(%q)", string(randRune))
		case "float32":
			randFloat, err := fuzzConsumer.GetFloat32()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "float32(%f)", randFloat)
		case "float64":
			randFloat, err := fuzzConsumer.GetFloat64()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "float64(%f)", randFloat)
		case "bool":
			randBool, err := fuzzConsumer.GetBool()
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(&b, "bool(%t)", randBool)
		default:
			panic(fmt.Sprintf("fuzzer uses unsupported type: %s", arg))
		}
		if argNumber != len(args)-1 {
			fmt.Fprintln(&b, "")
		}
	}
	return b.String()
}
