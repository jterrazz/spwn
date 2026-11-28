package machinefile_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: a mixin named name, its body after its header.
func kit(name, body string) string {
	return "machinefile: station.machinefile/v1\nname: " + name + "\n" + body
}

func files(named map[string]string) map[string]string {
	out := map[string]string{}
	for name, body := range named {
		out["machinefiles/"+name+".machine.yaml"] = body
	}
	return out
}

type scriptSaid struct{ phase, file, lines string }

func scriptsOf(r machinefile.Resolved) []scriptSaid {
	out := make([]scriptSaid, 0, len(r.Scripts))
	for _, s := range r.Scripts {
		out = append(out, scriptSaid{string(s.Phase), s.Origin.Name, s.Lines})
	}
	return out
}

func TestEachFilesRunLinesAreAStepOfTheirOwnInTheChainsOrder(t *testing.T) {
	top := machine("top", "{ os: ubuntu }", "[]")
	for _, tc := range []struct {
		how   string
		files map[string]string
		want  []scriptSaid
	}{
		{"two kits and the machine each say run.boot", map[string]string{
			"a":   kit("a", "run: { boot: echo a }\n"),
			"b":   kit("b", "run: { boot: echo b, image: make b }\n"),
			"top": top + "extends: [a, b]\nrun: { boot: echo top }\n",
		}, []scriptSaid{{"image", "b", "make b"}, {"boot", "a", "echo a"}, {"boot", "b", "echo b"}, {"boot", "top", "echo top"}}},
		{"a diamond reads the kit both extend once", map[string]string{
			"d":   kit("d", "run: { boot: echo d }\n"),
			"b":   kit("b", "extends: [d]\nrun: { boot: echo b }\n"),
			"c":   kit("c", "extends: [d]\nrun: { boot: echo c }\n"),
			"top": top + "extends: [b, c]\n",
		}, []scriptSaid{{"boot", "d", "echo d"}, {"boot", "b", "echo b"}, {"boot", "c", "echo c"}}},
		{"a file that says no phase adds none", map[string]string{
			"a":   kit("a", "apps: [fd]\n"),
			"top": top + "extends: [a]\nrun: { upgrade: echo up }\n",
		}, []scriptSaid{{"upgrade", "top", "echo up"}}},
	} {
		t.Run(tc.how, func(t *testing.T) {
			r, err := machinefile.Resolve(library(t, files(tc.files), nil), "top", machinefile.Machine{}, machinefile.Lock{})
			if err != nil {
				t.Fatal(err)
			}
			if got := scriptsOf(r); !slices.Equal(got, tc.want) {
				t.Errorf("scripts %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEachFilesRunLinesAreAValueOfTheirOwnInExplain(t *testing.T) {
	lib := library(t, files(map[string]string{
		"a":   kit("a", "run: { boot: echo a }\n"),
		"top": machine("top", "{ os: ubuntu }", "[]") + "extends: [a]\nrun: { boot: echo top }\n",
	}), nil)
	r, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	var got []scriptSaid
	for _, v := range r.Values {
		if v.Key == "run.boot" {
			got = append(got, scriptSaid{v.Key, v.Origin.Name, v.Text})
		}
	}
	if want := []scriptSaid{{"run.boot", "a", "echo a"}, {"run.boot", "top", "echo top"}}; !slices.Equal(got, want) {
		t.Errorf("run.boot values %+v, want %+v", got, want)
	}
}
