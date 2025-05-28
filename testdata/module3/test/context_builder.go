package test

import (
	"testing"
)

type StringGenerator func(n int) string

type Builder struct {
	T *testing.T
}