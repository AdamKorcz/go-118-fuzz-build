package submodule1

import (
	"testing"
	testpkg "module3/test"
)

func FuzzProcessItem(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte){
		builder := &testpkg.Builder{
			T:     t,
		}
		_ = builder
	})
}