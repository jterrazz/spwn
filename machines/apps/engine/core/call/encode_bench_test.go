package call_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

type benchRow struct {
	Name   string            `json:"name"`
	State  string            `json:"state"`
	CPUs   int               `json:"cpus"`
	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
	Inner  benchInner        `json:"inner"`
	Rows   []benchInner      `json:"rows"`
}

type benchInner struct {
	A string  `json:"a"`
	B int     `json:"b"`
	C float64 `json:"c"`
	D []int   `json:"d"`
}

func benchValue() benchRow {
	r := benchRow{Name: "linux-1", State: "running", CPUs: 4, Tags: []string{"a", "b", "c"}, Labels: map[string]string{"x": "1", "y": "2"}}
	for i := range 20 {
		r.Rows = append(r.Rows, benchInner{A: "row", B: i, C: 1.5, D: []int{1, 2, 3}})
	}
	return r
}

// invariant: a value that holds no error is encoded as json.Marshal
// encodes it, its type walked once per process and never per call.
func TestAnErrorFreeValueEncodesAtTheCostOfItsJSON(t *testing.T) {
	v := benchValue()
	if _, err := call.Encode(v, nil); err != nil {
		t.Fatal(err)
	}
	marshal := testing.AllocsPerRun(100, func() { _, _ = json.Marshal(v) })
	encode := testing.AllocsPerRun(100, func() { _, _ = call.Encode(v, nil) })
	if encode > marshal+1 {
		t.Errorf("Encode allocates %.0f times, json.Marshal %.0f", encode, marshal)
	}
}

func BenchmarkEncode(b *testing.B) {
	v := benchValue()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := call.Encode(v, nil); err != nil {
			b.Fatal(err)
		}
	}
}
