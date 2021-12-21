# go-118-fuzz-build

To use
```bash
# install gotip
git clone https://github.com/AdamKorcz/go-118-fuzz-build
cd go-118-fuzz-build
mv your_fuzzer.go ./
go run main.go -o fuzzer.a -func FuzzFoo .
clang -o compiled_fuzzer fuzzer.a -fsanitize=fuzzer
```
