# go-118-fuzz-build

## Disclaimer

This project is in its early stages and under heavy development. It is not ready for use yet and the code base is still largely experimental.

## To use
```bash
# Prepare your file
go install github.com/AdamKorcz/go-118-fuzz-build/addimports@latest
cd your/pkg
mv your_fuzzer_test.go your_fuzzer.go
addimports -path your_fuzzer.go

# Build the fuzzer
go install github.com/AdamKorcz/go-118-fuzz-build@latest
go-118-fuzz-build -o fuzzer.a -func FuzzFoo .
clang -o compiled_fuzzer fuzzer.a -fsanitize=fuzzer
```

## Corpus conversion utility

```bash
# Prepare your file
go install github.com/AdamKorcz/go-118-fuzz-build/addimports@latest
cd your/pkg
mv your_fuzzer_test.go your_fuzzer.go
addimports -path your_fuzzer.go

# Build the tool
go-118-fuzz-build -corpus-util -o fuzzutil -func FuzzFoo .

# Convert a libfuzzer corpus file to a native Go corpus file
cat libfuzzer-corpus-file | ./fuzzutil lib2go > testdata/fuzz/FuzzFoo/native-go-corpus-file
```