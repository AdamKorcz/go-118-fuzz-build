package module1

import (
	"fmt"
	"testing"
	"module1/submodule1"
)

var (
	b = submodule1.AA
)

func FuzzTest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) < 3 {
			return
		}
		if string(data[0]) == "a" {
			if string(data[1]) == "b" {
				if string(data[2]) == "c" {
					fmt.Println("b is: ", b)
				}
			}
		}
	})
}
