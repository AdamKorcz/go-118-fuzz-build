package fuzztesting

import (
	"github.com/AdamKorcz/go-118-fuzz-build/utils"
	"testing"
	"vitess.io/vitess/go/vt/sqlparser"
)

func FuzzFoo(f *utils.F) {
	f.Fuzz(func(t *testing.T, data, data2 []byte, data3 string, iint int8, iint2 uint16) {
		_, _ = sqlparser.Parse(string(data))
	})
}