package activity

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

func TestAMemoryReadingIsKeptToTheByte(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var s Series
	for i := range 30 {
		s.Add(Sample{At: from.Add(time.Duration(i) * time.Minute), CPUTotal: uint64(i) * 60_000, MemUsed: 1_181_116_006, SwapUsed: 123_456_789})
	}
	now := from.Add(29 * time.Minute)
	for _, x := range s.History(now) {
		if x.MemUsed != 1_181_116_006 || x.SwapUsed != 123_456_789 {
			t.Fatalf("a minute of the history reads %d B of memory and %d of swap; want 1181116006 and 123456789, as measured", x.MemUsed, x.SwapUsed)
		}
	}
	m := s.Measures(time.Hour, now)
	last := m.Memory[len(m.Memory)-1]
	if last.Mean != 1_181_116_006 || last.Max != 1_181_116_006 {
		t.Errorf("the hour's last memory bar is %+v; want its mean and max the very bytes measured", last)
	}
}

func TestTheRatesOfTheLastTenSecondsAreTheCountersOwn(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var s Series
	points := make([]Sample, 0, 8)
	for i := range 8 {
		x := Sample{At: from.Add(time.Duration(2*i) * time.Second), CPUTotal: uint64(i) * 2000, CPUBusy: uint64(i) * 797, NetIn: uint64(i) * 2_000_006}
		points = append(points, x)
		s.Add(x)
	}
	now := points[7].At
	got, window := Recent(&s, now)
	want := Between(points[2], points[7])
	if got != want || window != 10*time.Second {
		t.Fatalf("the last 10 s rate %+v over %v; want %+v over 10 s, the two samples' own counters", got, window, want)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"CPU":39.85,"DiskRead":0,"DiskWrite":0,"NetIn":1000003,"NetOut":0}` {
		t.Errorf("the rates are written %s; want the CPU to 2 decimals, the bytes a second whole", data)
	}
}

var (
	quoted     = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	tooPrecise = regexp.MustCompile(`\d\.\d{4,}`)
)

func TestNoNumberOfAnActivityAnswerCarriesMoreThanThreeDecimals(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(2*time.Hour, WatchedEvery, func(at time.Time) float64 { return 100.0 / 3 * float64(at.Second()%3) })
	now := sp.at
	last := sp.last
	last.Load = [3]float64{1.5302734375, 2.0009765625, 0.1}
	chart, measures := sp.s.Chart(time.Hour, now), sp.s.Measures(time.Hour, now)
	rates, window := Recent(&sp.s, now)
	r := Report{
		Source: FromEngine, Since: from, Rates: rates, Window: window, Chart: &chart, Measures: &measures,
		Last: Reading{
			Sample: last,
			Top:    Busiest{CPU: []Process{{PID: 1, Name: "go", Value: 100.0 / 7}}, Memory: []Process{{PID: 2, Name: "node", Value: 123456789}}},
			Run:    &Usage{CPU: 2.0 / 3, Memory: 1},
			VMs:    []VMUsage{{VM: "linux-1", Usage: Usage{CPU: 9.123456789, Memory: 5}}},
		},
		History: sp.s.History(now), Graph: sp.s.Graph(time.Hour, now),
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if found := tooPrecise.FindAllString(quoted.ReplaceAllString(string(data), `""`), 5); len(found) > 0 {
		t.Errorf("an activity answer writes %v; want every float rounded to what its reader needs", found)
	}
	if !regexp.MustCompile(`"vm":"linux-1","cpu":9.12,`).Match(data) {
		t.Errorf("a VM's usage is written %s; want its name beside its CPU", data)
	}
}
