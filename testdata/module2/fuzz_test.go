package module2

import (
	"testing"

	"module2/submodule1"
)

func FuzzA(f *testing.F) {
	f.Fuzz(func(t *testing.T, param string) {
		if param == submodule1.A {
			t.Fatal("Got the right one")
		}
	})
}