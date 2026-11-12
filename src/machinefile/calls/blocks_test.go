package calls

import (
	"context"
	"errors"
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

func depsOver(z *zshrcText) Deps {
	return Deps{Zshrc: z, Home: "/Users/a", Machine: machinefile.Machine{System: catalog.SystemSpec{OS: catalog.OSMacOS, Arch: catalog.ArchARM64}}}
}

func TestABlockIsJudgedAgainstWhatTheFileHolds(t *testing.T) {
	it, cat := libItem(t)
	plan := planOf(it, map[string]catalog.OptionValue{"port": "80"})
	body := "export LIB=/Users/a:80\n"
	cases := []struct {
		name      string
		file      string
		condition machinefile.Condition
		action    machinefile.Action
	}{
		{"absent is set", "mine\n", machinefile.Absent, machinefile.Set},
		{"as written is kept", "# >>> station:lib/path >>>\n" + body + "# <<< station:lib/path <<<\n", machinefile.Present, machinefile.Keep},
		{"edited by hand is left", "# >>> station:lib/path >>>\nexport LIB=mine\n# <<< station:lib/path <<<\n", machinefile.Changed, machinefile.Leave},
		{"damaged is left", "# >>> station:lib/path >>>\n" + body, machinefile.Changed, machinefile.Leave},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := withBlocks(t.Context(), depsOver(&zshrcText{c.file}), plan, cat)
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

const solo = `item: station.catalog/v1
kind: app
name: solo
description: Alone
provides: [solo]
methods:
    - os: macos
      script: { run: install-solo, check: solo --version }
      shell:
          - id: station:solo
            zsh: export SOLO={{ home }}
`

func TestTheBlockOfAnAppThePlanCarriesNoLongerIsRemovedWhileItIsStationsAlone(t *testing.T) {
	it, err := catalog.Decode("items/libs/solo.yaml", []byte(solo))
	if err != nil {
		t.Fatal(err)
	}
	cat := catalog.Catalog{Items: []catalog.Placed{{Item: it}}}
	block := func(body string) string { return "mine\n# >>> station:solo >>>\n" + body + "# <<< station:solo <<<\n" }
	cases := []struct {
		name, file string
		blocks     int
		action     machinefile.Action
	}{
		{"as station wrote it", block("export SOLO=/Users/a\n"), 1, machinefile.Remove},
		{"edited by hand", block("export SOLO=mine\n"), 1, machinefile.Leave},
		{"of no item the catalog knows", "# >>> station:other >>>\nx\n# <<< station:other <<<\n", 0, ""},
		{"none", "mine\n", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := withBlocks(t.Context(), depsOver(&zshrcText{c.file}), machinefile.Plan{}, cat)
			if err != nil || len(p.Blocks) != c.blocks {
				t.Fatalf("judged %+v, %v", p.Blocks, err)
			}
			if c.blocks == 1 && p.Blocks[0].Action != c.action {
				t.Errorf("judged %+v, want %s", p.Blocks[0], c.action)
			}
		})
	}
}
