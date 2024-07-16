package module1

import (
	"testing"
	"module1/submodule1"
)

var(
	b = submodule1.AA
)

func FuzzTest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = b
	})
}