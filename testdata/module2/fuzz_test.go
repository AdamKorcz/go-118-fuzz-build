package module2

import (
	"fmt"
	"testing"

	"module2/submodule3"
)

func FuzzTest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		ourVar := submodule3.SmallFunc(data)
		if ourVar == "B" {
			fmt.Println("We got B")
		}
	})
}
