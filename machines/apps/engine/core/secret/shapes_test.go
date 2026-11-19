package secret

import (
	"strings"
	"testing"
)

func TestTheUserinfoScanOfAWordOfManyURLsIsLinear(t *testing.T) {
	text := `{"links":[` + strings.Repeat(`{"url":"https://x.dev/a","by":"a@b.com"},`, 40000) + `{}]}`
	r := testing.Benchmark(func(b *testing.B) {
		for b.Loop() {
			credentials(text)
		}
	})
	if bound := int64(1e9); r.NsPerOp() > bound {
		t.Errorf("scanning %d bytes of %d URLs took %d ns, past %d", len(text), 40000, r.NsPerOp(), bound)
	}
}
