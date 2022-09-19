//go:build go1.18

package compact

import "github.com/AdamKorcz/go-118-fuzz-build/utils"

func FuzzFail(f *utils.F) {
	f.Fuzz(func(t *utils.T) {
		t.Error()
	})
}
