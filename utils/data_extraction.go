package utils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

/*
x is the func declaration node
src is the source code of the fuzzer
*/
func isFuzzFunc(x *ast.FuncDecl, src []byte) bool {
	for _, field := range x.Type.Params.List {
		argType := getStringVersion(field.Type.Pos(), field.Type.End(), src)
		if argType=="*testing.F" {
			return true
		}
	}
	return false
}

// Checks if the call expr is the inner fuzzer
func isInnerFuzzFunc(x *ast.CallExpr, fuzzIdentifier string, src []byte) bool {
	fuzzName := fmt.Sprintf("%s.Fuzz", fuzzIdentifier)
	currentFuncName := getStringVersion(x.Fun.Pos(), x.Fun.End(), src)
	if currentFuncName==fuzzName {
		return true
	}
	return false
}

func getStringVersion(start, end token.Pos, src  []byte) string {
    return string(src[start-1:end-1])
}


// Gets the args and body of f.Fuzz()
// params:
// x: The AST node of f.Fuzz()
// src: the source code of f.Fuzz()
func getFuzzerArgs(x *ast.CallExpr, src []byte) map[string]string {
	argMap := make(map[string]string)
	fuzzer := x.Args[0]
    arg0String := getStringVersion(fuzzer.Pos(), fuzzer.End(), src)
    body, err := innerFuzzBody([]byte(arg0String))
    if err != nil {
    	panic(err)
    }
	ast.Inspect(body, func(n ast.Node) bool {
	    if n==nil {
	        return false
	    }
	    switch x := n.(type) {
	    case *ast.FuncLit:
	    	// we have reached the function argument to f.Fuzz(): func(t *testing.T...)
	    	// we now check the params of it

	    	// the "t *testing.T" param:
	    	testerArg := x.Type.Params.List[0]
	    	testerArgName := testerArg.Names[0].Name
	    	testArgType := getStringVersion(testerArg.Type.Pos(), testerArg.Type.End(), []byte(arg0String))
	    	_, _ = testArgType, testerArgName

	    	// parameters after "*testing.T":
	    	dataParams := x.Type.Params.List[1:]
	    	for _, dataArg := range dataParams {

		    	// print the names of each parameter. 
		    	// This is for cases like this:
		    	// data1, data2 []byte.
		    	for _, name := range dataArg.Names {
		    		dataArgType := getStringVersion(dataArg.Type.Pos(), dataArg.Type.End(), []byte(arg0String))
		    		argMap[name.Name] = dataArgType
		    	}

	    	}
	    }
	    return true
	})
	return argMap
}

func parseFuzzer(filename string) ([]byte, *ast.File, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return []byte("nil"), nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return []byte("nil"), nil, err
	}
	return src, f, nil
}

// Finds the FuzzXXX(f *testing.F) function and returns it as []byte
func mainFuzzBody(f *ast.File, src []byte) ([]byte, error) {
	var body []byte


	ast.Inspect(f, func(n ast.Node) bool {		
	    if n==nil {
	        return false
	    }
	    switch x := n.(type) {
		case *ast.FuncDecl:
			if isFuzzFunc(x, src) {
				bodyStr := getStringVersion(x.Body.Lbrace, x.Body.Rbrace, src)
				body = []byte(bodyStr)
				return true
			} else {
				fmt.Println("Not a fuzzer")
			}
		}
		return true
	})
	if body == nil {
		return body, fmt.Errorf("Could not get fuzzer")
	}
	return body[1:], nil
}


// innerFuzzBody parses func FuzzXXX(f *testing.F)
func innerFuzzBody(src []byte) (ast.Expr, error) {
	expr, err := parser.ParseExpr(string(src))
	if err != nil {
		panic(err)
	}
	return expr, nil
}

func fuzzerArgs(f *ast.File, src []byte) map[string]string {
	// Get the main fuzz function (func FuzzXXX(f *testing.F))
	mainFuzzBody, err := mainFuzzBody(f, src)
	if err != nil {
		panic(err)
	}

	// parse the main fuzz body
	innerFuzzBody, err := innerFuzzBody(mainFuzzBody)
	if err != nil {
		panic(err)
	}

	// Get the "any" args of f.Fuzz(func(t *testing.T, any))
	args := getArgs(innerFuzzBody, mainFuzzBody)

	return args
}

// returns the args of the func(t *testing.T, any)
func getArgs(expr ast.Expr, body2 []byte) map[string]string {
	var args map[string]string
	ast.Inspect(expr, func(n2 ast.Node) bool {
		switch x2 := n2.(type) {
		case *ast.CallExpr:

			// TODO: This should be checked in the fuzzer:
			fuzzIdentifier := "f"

			// check if it is f.Fuzz() func
			if !isInnerFuzzFunc(x2, fuzzIdentifier, body2) {
				return false
			}
			args = getFuzzerArgs(x2, body2)
		}
		return true
	})
	return args
}

/////////////////////////////////////////
// turning the args into code
/////////////////////////////////////////

// gets a declaration for a byte slice
func getByteSlice(key string) string {
	var byteSlice strings.Builder
	byteSlice.WriteString(fmt.Sprintf("\t%s, err := fuzzConsumer.GetBytes()\n", key))
	byteSlice.WriteString("\tif err != nil { return }\n")
	return byteSlice.String()
}

// gets a declaration for a string
func getString(key string) string {
	var stringDeclaration strings.Builder
	stringDeclaration.WriteString(fmt.Sprintf("\t%s, err := fuzzConsumer.GetString()\n", key))
	stringDeclaration.WriteString("\tif err != nil { return }\n")
	return stringDeclaration.String()

}

// gets declarations based on the type of the argument
func getDeclarations(m map[string]string) string {
	var declarations strings.Builder
	for k, v := range m {
		switch v {
		case "[]byte":
			declarations.WriteString(getByteSlice(k))
		case "string":
			declarations.WriteString(getString(k))
		}
    }
    return declarations.String()
}

func getParameters(m map[string]string) string {
	var parameters strings.Builder
	i := 0
	for k, _ := range m {
		if i==0 {
			parameters.WriteString(k)
		}else{
			parameters.WriteString(", "+k)
		}
		i++
	}
	return parameters.String()
}

func GetParamDeclarations(file string) string {
	src, f, err := parseFuzzer(file)
	if err != nil {
		panic(err)
	}
	
    fuzzerArgs := fuzzerArgs(f, src)

    declarations := getDeclarations(fuzzerArgs)
    return declarations
}

func GetParams(file string) string {
	src, f, err := parseFuzzer(file)
	if err != nil {
		panic(err)
	}
    fuzzerArgs := fuzzerArgs(f, src)

    parameters := getParameters(fuzzerArgs)
    return parameters
}