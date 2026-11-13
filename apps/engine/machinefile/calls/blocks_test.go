package calls

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

type zshrcText struct{ text string }

func (z *zshrcText) Read(context.Context) (string, error) { return z.text, nil }
func (z *zshrcText) Edit(_ context.Context, edit func(string) (string, error)) error {
	text, err := edit(z.text)
	if err == nil {
		z.text = text
	}
	return err
}

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
	return Deps{Zshrc: z, Ledger: &writtenLedger{blocks: slices.Clone(written)}, Home: "/Users/a", Machine: machinefile.Machine{System: catalog.SystemSpec{OS: catalog.OSMacOS, Arch: catalog.ArchARM64}}}
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

// value: an extra of the plan whose app reads as seen, by the method the
// ledger recorded, "pkg".
func extraSeen(item string, seen machinefile.Seen, action machinefile.Action, requiredBy ...string) machinefile.ExtraLine {
	return machinefile.ExtraLine{
		Installed:  machinefile.Installed{Item: item, Method: "pkg", Machinefile: "laptop"},
		Line:       machinefile.Line{Step: machinefile.Step{Item: item, Kind: catalog.KindApp, Method: "pkg"}, Seen: seen, Action: action},
		RequiredBy: requiredBy,
	}
}

var (
	soloTaken     = extraSeen("solo", machinefile.Seen{Condition: machinefile.Present, Method: "pkg"}, machinefile.Remove)
	soloGone      = extraSeen("solo", machinefile.Seen{Condition: machinefile.Absent}, machinefile.Remove)
	soloYours     = extraSeen("solo", machinefile.Seen{Condition: machinefile.Present, Method: "other"}, machinefile.Leave)
	soloRequired  = extraSeen("solo", machinefile.Seen{Condition: machinefile.Present, Method: "pkg"}, machinefile.Remove, "app")
	soloRecord    = machinefile.Written{ID: "station:solo", Item: "solo", Digest: machinefile.BlockDigest("export SOLO=1\n")}
	soloFile      = func(body string) string { return "mine\n# >>> station:solo >>>\n" + body + "# <<< station:solo <<<\n" }
	soloUntouched = soloFile("export SOLO=1\n")
)

func TestABlockStationWroteForAnExtraGoesWithItsApp(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		written []machinefile.Written
		extra   []machinefile.ExtraLine
		blocks  int
		action  machinefile.Action
		fate    machinefile.Fate
	}{
		{"its app taken back", soloUntouched, []machinefile.Written{soloRecord}, []machinefile.ExtraLine{soloTaken}, 1, machinefile.Remove, machinefile.Taken},
		{"its app already gone", soloUntouched, []machinefile.Written{soloRecord}, []machinefile.ExtraLine{soloGone}, 1, machinefile.Remove, machinefile.Gone},
		{"its app the owner's since", soloUntouched, []machinefile.Written{soloRecord}, []machinefile.ExtraLine{soloYours}, 1, machinefile.Leave, machinefile.KeptAnotherWay},
		{"its app still required", soloUntouched, []machinefile.Written{soloRecord}, []machinefile.ExtraLine{soloRequired}, 1, machinefile.Leave, machinefile.KeptRequired},
		{"edited by hand since", soloFile("export SOLO=mine\n"), []machinefile.Written{soloRecord}, []machinefile.ExtraLine{soloTaken}, 1, machinefile.Leave, machinefile.KeptEdited},
		{"its app no extra: adopted, another Machinefile's, skipped here", soloUntouched, []machinefile.Written{soloRecord}, nil, 0, "", ""},
		{"never written by station", soloUntouched, nil, []machinefile.ExtraLine{soloTaken}, 0, "", ""},
		{"none in the file", "mine\n", []machinefile.Written{soloRecord}, []machinefile.ExtraLine{soloTaken}, 0, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := withBlocks(t.Context(), depsOver(&zshrcText{c.file}, c.written...), machinefile.Plan{Extra: c.extra}, catalog.Catalog{})
			if err != nil || len(p.Blocks) != c.blocks {
				t.Fatalf("judged %+v, %v", p.Blocks, err)
			}
			if b := p.Blocks; c.blocks == 1 && (b[0].Condition != machinefile.Extra || b[0].Action != c.action || b[0].Fate != c.fate || b[0].Item != "solo") {
				t.Errorf("judged %+v, want %s and %s", b[0], c.action, c.fate)
			}
		})
	}
}

func TestApplyRecordsTheBlockItWrites(t *testing.T) {
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
	extra := machinefile.Plan{Blocks: []machinefile.Block{{Item: "solo", ID: "station:solo", Zsh: body, Condition: machinefile.Extra, Action: machinefile.Remove, Fate: machinefile.Taken}}}
	if done := r.blocks(t.Context(), extra); len(done) != 0 {
		t.Errorf("a bare apply took %+v", done)
	}
}

func TestABlockOfStationsTextWithNoRecordOfItIsRecorded(t *testing.T) {
	body := "export SOLO=1\n"
	file := "# >>> station:solo >>>\n" + body + "# <<< station:solo <<<\n"
	keep := machinefile.Block{Item: "solo", ID: "station:solo", Zsh: body, Condition: machinefile.Present, Action: machinefile.Keep}
	cases := []struct {
		name    string
		written []machinefile.Written
	}{
		{"with no record", nil},
		{"with the record of other text", []machinefile.Written{{ID: "station:solo", Item: "solo", Digest: machinefile.BlockDigest("export SOLO=0\n")}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			z, l := &zshrcText{file}, &writtenLedger{blocks: slices.Clone(c.written)}
			r := &run{d: Deps{Zshrc: z, Ledger: l, Now: epoch}, there: map[string]bool{"solo": true}}
			if done := r.blocks(t.Context(), machinefile.Plan{Blocks: []machinefile.Block{keep}}); len(done) != 1 || done[0].State != catalog.StateOK {
				t.Fatalf("kept %+v", done)
			}
			if len(l.blocks) != 1 || l.blocks[0].Digest != machinefile.BlockDigest(body) || l.blocks[0].Item != "solo" {
				t.Errorf("recorded %+v", l.blocks)
			}
			if z.text != file {
				t.Errorf("rewrote the file: %q", z.text)
			}
		})
	}
}

func TestRemoveTakesABlockOnlyWithItsApp(t *testing.T) {
	cases := []struct {
		name      string
		file      string
		extra     machinefile.ExtraLine
		broken    bool
		detail    string
		left      string
		recorded  int
		appForgot bool
	}{
		{"its app taken back", soloUntouched, soloTaken, false, "removed", "mine\n", 0, true},
		{"its app already gone", soloUntouched, soloGone, false, "removed", "mine\n", 0, true},
		{"its app failed to uninstall", soloUntouched, soloTaken, true, "kept with solo: judged again next run", soloUntouched, 1, false},
		{"its app still required", soloUntouched, soloRequired, false, "kept with solo: judged again next run", soloUntouched, 1, false},
		{"its app the owner's since", soloUntouched, soloYours, false, "kept with solo, and yours", soloUntouched, 0, true},
		{"edited by hand, its app still required", soloFile("export SOLO=mine\n"), soloRequired, false, editedKept, soloFile("export SOLO=mine\n"), 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			z := &zshrcText{c.file}
			l := &writtenLedger{blocks: []machinefile.Written{soloRecord}}
			d := depsOver(z)
			d.Ledger = l
			calls := &calls{}
			if c.broken {
				calls.broken = []string{"solo"}
			}
			d.Calls, d.Now = calls, epoch
			p, err := withBlocks(t.Context(), d, machinefile.Plan{Extra: []machinefile.ExtraLine{c.extra}}, catalog.Catalog{})
			if err != nil {
				t.Fatal(err)
			}
			r := &run{d: d, name: "laptop", ended: map[string]catalog.State{}}
			_, blocks := r.remove(t.Context(), p)
			if len(blocks) != 1 || blocks[0].Detail != c.detail || blocks[0].State != catalog.StateOK {
				t.Fatalf("blocks %+v, want %q", blocks, c.detail)
			}
			if z.text != c.left || len(l.blocks) != c.recorded || slices.Contains(l.deleted, "solo") != c.appForgot {
				t.Errorf("left %q, records %+v, app forgotten %v", z.text, l.blocks, l.deleted)
			}
		})
	}
}

// value: a ~/.zshrc another writer changes, file and ledger, in the lock's
// turn before this edit gets it.
type racedZshrc struct {
	zshrcText
	first func(*zshrcText)
}

func (z *racedZshrc) Edit(ctx context.Context, edit func(string) (string, error)) error {
	if z.first != nil {
		z.first(&z.zshrcText)
		z.first = nil
	}
	return z.zshrcText.Edit(ctx, edit)
}

func TestABlockIsJudgedInsideTheEditAgainstTheLedgerAsItIsThen(t *testing.T) {
	older, body := "export SOLO=0\n", "export SOLO=1\n"
	l := &writtenLedger{}
	z := &racedZshrc{zshrcText: zshrcText{"mine\n"}, first: func(z *zshrcText) {
		z.text = soloFile(older)
		l.blocks = []machinefile.Written{{ID: "station:solo", Item: "solo", Digest: machinefile.BlockDigest(older)}}
	}}
	r := &run{d: Deps{Zshrc: z, Ledger: l, Now: epoch}, there: map[string]bool{"solo": true}}
	set := machinefile.Block{Item: "solo", ID: "station:solo", Zsh: body, Condition: machinefile.Absent, Action: machinefile.Set}
	done := r.blocks(t.Context(), machinefile.Plan{Blocks: []machinefile.Block{set}})
	if len(done) != 1 || done[0].Detail != "written" || z.text != soloFile(body) {
		t.Errorf("wrote %+v, left %q; want station's older text, recorded by the other writer, rewritten", done, z.text)
	}
}

func TestABlockTheOwnerDeletedIsToldApartFromOneHeEdited(t *testing.T) {
	z := &zshrcText{soloUntouched}
	d := depsOver(z)
	l := &writtenLedger{blocks: []machinefile.Written{soloRecord}}
	d.Ledger, d.Calls, d.Now = l, &calls{}, epoch
	p, err := withBlocks(t.Context(), d, machinefile.Plan{Extra: []machinefile.ExtraLine{soloTaken}}, catalog.Catalog{})
	if err != nil {
		t.Fatal(err)
	}
	z.text = "mine\n"
	_, blocks := (&run{d: d, name: "laptop", ended: map[string]catalog.State{}}).remove(t.Context(), p)
	if len(blocks) != 1 || blocks[0].Detail != deletedGone || blocks[0].State != catalog.StateOK || len(l.blocks) != 0 {
		t.Errorf("blocks %+v, records %+v; want %q and no record", blocks, l.blocks, deletedGone)
	}
}

func TestABlockAlreadyThereIsRecordedOnlyOnceItsAppIs(t *testing.T) {
	body := "export SOLO=1\n"
	z, l := &zshrcText{soloFile(body)}, &writtenLedger{}
	r := &run{d: Deps{Zshrc: z, Ledger: l, Now: epoch}, there: map[string]bool{}}
	keep := machinefile.Block{Item: "solo", ID: "station:solo", Zsh: body, Condition: machinefile.Present, Action: machinefile.Keep}
	done := r.blocks(t.Context(), machinefile.Plan{Blocks: []machinefile.Block{keep}})
	if len(done) != 1 || done[0].State != catalog.StateAbsent || len(l.blocks) != 0 {
		t.Errorf("kept %+v, recorded %+v; want it waiting on solo, unrecorded", done, l.blocks)
	}
}
