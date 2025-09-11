module github.com/AdamKorcz/go-118-fuzz-build/cmd/convertLibFuzzerTestcaseToStdLibGo

go 1.25.0

replace github.com/AdamKorcz/go-118-fuzz-build => ../..

require (
	github.com/AdamKorcz/go-118-fuzz-build v0.0.0-00010101000000-000000000000
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	golang.org/x/tools v0.36.0
)

require (
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
)
