package automation

import "testing"

// must is the universal "fail the test on error" helper used across
// every test file in this package. Kept short so call sites stay
// flat and readable: `must(t, store.RecordFire(...))`.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
