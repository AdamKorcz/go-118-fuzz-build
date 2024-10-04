package module2

import (
	"fmt"
	"testing"

	"module2/submodule1"
)

func FuzzTest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) < 3 {
			return
		}
		if string(data[0]) == "a" {
			if string(data[1]) == "b" {
				if string(data[2]) == "b" {
					fmt.Println("b is: ", submodule1.A)
				}
			}
		}
	})
}
