package utils

import "testing"

func TestSkip(t *testing.T) {
	f := NewF(t.Name(), []byte{})
	f.Fuzz(func(dut *T) {
		dut.Skip()
		t.Error("Should not be reached")
	})
	if !f.Skipped() {
		t.Error("Did not skip")
	}
}

func TestError(t *testing.T) {
	f := NewF(t.Name(), []byte{})
	var ok bool
	defer func() {
		if recover() == nil {
			t.Error("Did not panic")
		}
		if !f.Failed() {
			t.Error("Did not fail")
		}
		if !ok {
			t.Error("Test did not continue")
		}
	}()
	f.Fuzz(func(dut *T) {
		dut.Error()
		ok = true
	})
}

func TestFatal(t *testing.T) {
	f := NewF(t.Name(), []byte{})
	defer func() {
		if recover() == nil {
			t.Error("Did not panic")
		}
		if !f.Failed() {
			t.Error("Did not fail")
		}
	}()
	f.Fuzz(func(dut *T) {
		dut.Fatal()
		t.Fatal("Should not be reached")
	})
}

func TestSubtest(t *testing.T) {
	f := NewF(t.Name(), []byte{})
	var ok bool
	f.Fuzz(func(dut *T) {
		dut.Run("", func(*T) {
			ok = true
		})
	})
	if !ok {
		t.Error("Subtest did not run")
	}
}

func TestSubtestError(t *testing.T) {
	f := NewF(t.Name(), []byte{})
	defer func() {
		if recover() == nil {
			t.Error("Did not panic")
		}
		if !f.Failed() {
			t.Error("Did not fail")
		}
	}()
	f.Fuzz(func(dut *T) {
		dut.Run("", func(dut *T) {
			dut.Error()
		})
	})
}

func TestCleanup(t *testing.T) {
	cases := map[string]func(*T){
		"Pass":  func(dut *T) {},
		"Skip":  func(dut *T) { dut.Skip() },
		"Fail":  func(dut *T) { dut.Fail() },
		"Panic": func(dut *T) { panic(nil) },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			f := NewF(t.Name(), []byte{})
			var ok bool
			func() {
				defer func() { _ = recover() }()
				f.Fuzz(func(dut *T) {
					dut.Cleanup(func() { ok = true })
					fn(dut)
				})
			}()
			if !ok {
				t.Error("Cleanup did not run")
			}
		})
	}
}
