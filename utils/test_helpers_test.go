package utils

import (
	"fmt"
	"testing"
)

func FuzzRune(f *testing.F) {
	f.Fuzz(func(t *testing.T, input rune) {
		fmt.Println(input)
	})
}