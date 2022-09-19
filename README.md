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

## Skip

Calling `t.Skip`, `t.Skipf`, or `t.SkipNow` will immediately terminate the test
and mark it as skipped. Skipping the top-level test will tell libfuzzer to
exclude that input from the corpus. Specifically, the fuzzer will return `-1`.
Skipping a sub-test has no effect other than immediately stopping the sub-test.

If github.com/AdaLogics/go-fuzz-headers cannot generate the requested values
from the input, that input will be skipped, telling libfuzzer to exclude that
input from the corpus.

## Fail

Calling `t.Fail`, `t.Error`, and `t.Errorf` will mark the test as failed.
Calling `t.FailNow`, `t.Fatal`, or `t.Fatalf` will immediately terminate the
test and mark it as failed. If a sub-test is marked as failed, its parent will
also be marked as failed. If a test fails the fuzzer will panic once the
top-level test is complete or terminated. Thus libfuzzer will mark inputs that
cause a test to fail as crashers.

## Limitations

`t.Helper`, `t.Parallel`, and `t.Setenv` have no effect. Since `t.Parallel` is
not honored, sub-tests are run synchronously.