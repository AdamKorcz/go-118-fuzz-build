#/bin/bash -eu


mkdir fuzz && cd fuzz
compile_native_go_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/les FuzzLesNative fuzzLesNative
compile_native_go_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/rlp FuzzRlpNative fuzzRlpNative
compile_native_go_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/runtime FuzzRuntimeNative fuzzVmRuntimeNative

