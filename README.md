# go-118-fuzz-build

## Disclaimer

This project is in its early stages and under heavy development. It is not ready for use yet and the code base is still largely experimental.

## To use
```bash
# install gotip
git clone https://github.com/AdamKorcz/go-118-fuzz-build
cd go-118-fuzz-build
mv your_fuzzer.go ./
gotip run main.go -o fuzzer.a -func FuzzFoo .
clang -o compiled_fuzzer fuzzer.a -fsanitize=fuzzer
```
