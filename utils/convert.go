package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// TODO: Once go-fuzz-headers implements the reverse conversion, use that to
// implement func NativeToLibFuzzer(fn func(*F))

// LibFuzzerToNative converts a libfuzzer corpus entry to Go's native corpus
// format.
func LibFuzzerToNative(fn func(*F), data []byte) ([]byte, bool) {
	fuzzer := NewCaptureF("LibFuzzerToNative", data)
	fn(fuzzer)

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "go test fuzz v1\n")
	for _, v := range fuzzer.values {
		switch v := v.(type) {
		case []byte:
			fmt.Fprintf(buf, "[]byte(%q)\n", v)
		case string:
			fmt.Fprintf(buf, "string(%q)\n", v)
		default:
			fmt.Fprintf(buf, "%T(%[1]v)\n", v)
		}
	}
	io.ReadAll(os.Stdin)
	return buf.Bytes(), fuzzer.Skipped()
}
