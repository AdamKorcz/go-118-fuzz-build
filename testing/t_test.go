package testing

import (
	"os"
	"testing"
)

func TestTempDirs(t *testing.T) {
	fuzzT := NewT()
	tDir1 := fuzzT.TempDir()

	fi, err := os.Stat(tDir1)
	if err != nil {
		panic(err)
	}
	if !fi.IsDir() {
		t.Fatal("This should be a directory")
	}

	tDir2 := fuzzT.TempDir()
	fi, err = os.Stat(tDir2)
	if err != nil {
		panic(err)
	}
	if !fi.IsDir() {
		t.Fatal("This should be a directory")
	}

	fuzzT.CleanupTempDirs()

	fi, err = os.Stat(tDir1)
	if err == nil {
		panic(err)
	}
	if fi != nil {
		t.Fatal("fi is not nil")
	}

	fi, err = os.Stat(tDir2)
	if err == nil {
		panic(err)
	}
	if fi != nil {
		t.Fatal("fi is not nil")
	}
}
