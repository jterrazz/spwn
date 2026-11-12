package calls

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
)

type zshrcText struct{ text string }

func (z *zshrcText) Read(context.Context) (string, error)    { return z.text, nil }
func (z *zshrcText) Write(_ context.Context, s string) error { z.text = s; return nil }

const lib = `item: station.catalog/v1
kind: app
name: lib
description: A library
provides: [lib]
options:
    - name: port
      label: Port
      type: integer
      required: true
methods:
    - os: macos
      script: { run: install-lib, check: lib --version }
      shell:
          - id: station:lib/path
            zsh: |
                export LIB={{ home }}:{{ option.port }}
`

func libItem(t *testing.T) (catalog.Item, catalog.Catalog) {
	t.Helper()
	it, err := catalog.Decode("items/libs/lib.yaml", []byte(lib))
	if err != nil {
		t.Fatal(err)
	}
	return it, catalog.Catalog{Items: []catalog.Placed{{Item: it}}}
}

func planOf(it catalog.Item, values map[string]catalog.OptionValue) machinefile.Plan {
	step := machinefile.Step{Item: it.Name, Kind: catalog.KindApp, Spec: it.Methods[0], Values: values}
	return machinefile.Plan{Lines: []machinefile.Line{{Step: step, Action: machinefile.Install}}}
}

// value: a ledger that holds the blocks it was given, and no install.
type writtenLedger struct {
	ledger
	blocks []machinefile.Written
}

func (l *writtenLedger) Blocks(context.Context) ([]machinefile.Written, error) { return l.blocks, nil }

func (l *writtenLedger) PutBlock(_ context.Context, w machinefile.Written) error {
	l.blocks = append(slices.DeleteFunc(l.blocks, func(e machinefile.Written) bool { return e.ID == w.ID }), w)
	return nil
}

func (l *writtenLedger) DeleteBlock(_ context.Context, id string) error {
	l.blocks = slices.DeleteFunc(l.blocks, func(e machinefile.Written) bool { return e.ID == id })
	return nil
}

func wrote(id, body string) machinefile.Written {
	return machinefile.Written{ID: id, Item: "lib", Digest: machinefile.BlockDigest(body)}
}

func depsOver(z *zshrcText, written ...machinefile.Written) Deps {
	return Deps{Zshrc: z, Ledger: &writtenLedger{blocks: written}, Home: "/Users/a", Machine: machinefile.Machine{System: catalog.SystemSpec{OS: catalog.OSMacOS, Arch: catalog.ArchARM64}}}
}

func TestABlockIsJudgedAgainstWhatTheFileHolds(t *testing.T) {
	it, cat := libItem(t)
	plan := planOf(it, map[string]catalog.OptionValue{"port": "80"})
	body := "export LIB=/Users/a:80\n"
	block := func(text string) string {
		return "# >>> station:lib/path >>>\n" + text + "# <<< station:lib/path <<<\n"
	}
	earlier := "export LIB=/Users/a:8080\n"
	cases := []struct {
		name      string
		file      string
		written   []machinefile.Written
		condition machinefile.Condition
		action    machinefile.Action
	}{
		{"absent is set", "mine\n", nil, machinefile.Absent, machinefile.Set},
		{"absent though written once is set again", "mine\n", []machinefile.Written{wrote("station:lib/path", body)}, machinefile.Absent, machinefile.Set},
		{"as written is kept", block(body), nil, machinefile.Present, machinefile.Keep},
		{"what station last wrote, written otherwise now, is outdated and set", block(earlier), []machinefile.Written{wrote("station:lib/path", earlier)}, machinefile.Outdated, machinefile.Set},
		{"edited by hand is left", block("export LIB=mine\n"), nil, machinefile.Changed, machinefile.Leave},
		{"edited by hand since station wrote it is left", block("export LIB=mine\n"), []machinefile.Written{wrote("station:lib/path", earlier)}, machinefile.Changed, machinefile.Leave},
		{"damaged is left", "# >>> station:lib/path >>>\n" + body, []machinefile.Written{wrote("station:lib/path", body)}, machinefile.Changed, machinefile.Leave},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := withBlocks(t.Context(), depsOver(&zshrcText{c.file}, c.written...), plan, cat)
			if err != nil || len(p.Blocks) != 1 {
				t.Fatalf("judged %+v, %v", p.Blocks, err)
			}
			if b := p.Blocks[0]; b.Condition != c.condition || b.Action != c.action || b.Zsh != body || b.ID != "station:lib/path" {
				t.Errorf("judged %+v, want %s and %s", b, c.condition, c.action)
			}
		})
	}
}

func TestAnOptionWithNoValueIsAPlanErrorNeverAnEmptyString(t *testing.T) {
	it, cat := libItem(t)
	_, err := withBlocks(t.Context(), depsOver(&zshrcText{}), planOf(it, nil), cat)
	var bad *machinefile.BadValueError
	if !errors.As(err, &bad) || bad.Key != "shell" {
		t.Errorf("planned with %v, want a bad value naming the shell", err)
	}
}

func TestABlockOfAnAppThePlanCarriesNoLongerIsExtraWhileItIsWhatStationLastWrote(t *testing.T) {
	block := func(body string) string { return "mine\n# >>> station:solo >>>\n" + body + "# <<< station:solo <<<\n" }
	written := []machinefile.Written{{ID: "station:solo", Item: "solo", Digest: machinefile.BlockDigest("export SOLO=1\n")}}
	cases := []struct {
		name, file string
		written    []machinefile.Written
		blocks     int
		condition  machinefile.Condition
		action     machinefile.Action
	}{
		{"as station last wrote it", block("export SOLO=1\n"), written, 1, machinefile.Extra, machinefile.Remove},
		{"edited by hand since", block("export SOLO=mine\n"), written, 1, machinefile.Changed, machinefile.Leave},
		{"never written by station", block("export SOLO=1\n"), nil, 0, "", ""},
		{"none in the file", "mine\n", written, 0, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := withBlocks(t.Context(), depsOver(&zshrcText{c.file}, c.written...), machinefile.Plan{}, catalog.Catalog{})
			if err != nil || len(p.Blocks) != c.blocks {
				t.Fatalf("judged %+v, %v", p.Blocks, err)
			}
			if c.blocks == 1 && (p.Blocks[0].Condition != c.condition || p.Blocks[0].Action != c.action || p.Blocks[0].Item != "solo") {
				t.Errorf("judged %+v, want %s and %s", p.Blocks[0], c.condition, c.action)
			}
		})
	}
}

func TestApplyRecordsTheBlockItWritesAndRemoveTakesItOnlyOnceItsAppGoes(t *testing.T) {
	body := "export SOLO=1\n"
	z, l := &zshrcText{"mine\n"}, &writtenLedger{}
	r := &run{d: Deps{Zshrc: z, Ledger: l, Now: epoch}, there: map[string]bool{"solo": true}}
	set := machinefile.Block{Item: "solo", ID: "station:solo", Zsh: body, Condition: machinefile.Absent, Action: machinefile.Set}
	if done := r.blocks(t.Context(), machinefile.Plan{Blocks: []machinefile.Block{set}}); len(done) != 1 || done[0].Detail != "written" {
		t.Fatalf("wrote %+v", done)
	}
	if len(l.blocks) != 1 || l.blocks[0].Digest != machinefile.BlockDigest(body) || l.blocks[0].Item != "solo" {
		t.Fatalf("recorded %+v", l.blocks)
	}
	stale := machinefile.Plan{Blocks: []machinefile.Block{{Item: "solo", ID: "station:solo", Zsh: body, Condition: machinefile.Extra, Action: machinefile.Remove}}}
	if done := r.blocks(t.Context(), stale); len(done) != 0 {
		t.Errorf("a bare apply took %+v", done)
	}
	written := z.text
	r.stays = map[string]bool{"solo": true}
	if done := r.removedBlocks(t.Context(), stale); len(done) != 1 || done[0].Detail != "kept with solo: judged again next run" || z.text != written {
		t.Errorf("an app that stays lost its block: %+v\n%s", done, z.text)
	}
	r.stays = map[string]bool{}
	if done := r.removedBlocks(t.Context(), stale); len(done) != 1 || done[0].Detail != "removed" || z.text != "mine\n" || len(l.blocks) != 0 {
		t.Errorf("removed %+v, file %q, record %+v", done, z.text, l.blocks)
	}
}
