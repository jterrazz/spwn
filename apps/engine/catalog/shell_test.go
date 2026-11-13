package catalog_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const pg = `package: station.package/v1
kind: app
name: pg
description: Database
options:
    - name: port
      label: Port
      type: integer
      default: 5432
    - name: token
      label: Token
      type: string
      secret: true
methods:
    - os: macos
      brew: pg
      shell:
          - id: station:pg/path
            zsh: |
                export PGPORT={{ option.port }}
                [[ -d {{ brew.prefix }}/opt/pg/bin ]] && path=({{ brew.prefix }}/opt/pg/bin $path)
`

func refusedAt(t *testing.T, file, body, want string) {
	t.Helper()
	_, err := catalog.Decode(file, []byte(body))
	var refusal *catalog.FileError
	if !errors.As(err, &refusal) || !strings.Contains(err.Error(), want) {
		t.Errorf("decoded with %v, want a refusal saying %q", err, want)
	}
}

func refusedPg(t *testing.T, body, want string) {
	t.Helper()
	refusedAt(t, "packages/databases/pg.yaml", body, want)
}

func shelled(t *testing.T, body string) catalog.Package {
	t.Helper()
	it, err := catalog.Decode("packages/databases/pg.yaml", []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func TestAMethodDecodesItsShellBlocks(t *testing.T) {
	it := shelled(t, pg)
	if got := it.Methods[0].Shell; len(got) != 1 || got[0].ID != "station:pg/path" {
		t.Errorf("decoded %+v", got)
	}
}

func TestAShellVariableIsFilledFromTheMachineAndTheOptions(t *testing.T) {
	it := shelled(t, pg)
	cases := []struct {
		name string
		vars catalog.ShellVars
		want string
	}{
		{"apple silicon with the default port", catalog.ShellVars{OS: catalog.OSMacOS, Arch: catalog.ArchARM64},
			"export PGPORT=5432\n[[ -d /opt/homebrew/opt/pg/bin ]] && path=(/opt/homebrew/opt/pg/bin $path)\n"},
		{"intel with a given port", catalog.ShellVars{OS: catalog.OSMacOS, Arch: catalog.ArchAMD64, Options: map[string]catalog.OptionValue{"port": "6000"}},
			"export PGPORT=6000\n[[ -d /usr/local/opt/pg/bin ]] && path=(/usr/local/opt/pg/bin $path)\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := it.RenderShell(it.Methods[0].Shell[0], c.vars)
			if err != nil || got != c.want {
				t.Errorf("rendered %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestAnUnknownShellVariableIsRefusedAtDecodeAndAtRender(t *testing.T) {
	refusedPg(t, strings.Replace(pg, "{{ option.port }}", "{{ brew.cellar }}", 1), "{{ brew.cellar }} is none of")
	refusedPg(t, strings.Replace(pg, "{{ option.port }}", "{{ option.nope }}", 1), "names no option of the package")
	refusedPg(t, strings.Replace(pg, "{{ option.port }}", "{{ option.token }}", 1), "is secret")
	refusedPg(t, strings.Replace(pg, "{{ option.port }}", "{{ {{ home }}", 1), "is no variable")
	it := catalog.Package{Name: "x"}
	if _, err := it.RenderShell(catalog.ShellBlock{ID: "station:x", Zsh: "echo {{ shell }}"}, catalog.ShellVars{}); err == nil {
		t.Error("an unknown name rendered as an empty string")
	}
	if _, err := it.RenderShell(catalog.ShellBlock{ID: "station:x", Zsh: "echo {{ option.port }}"}, catalog.ShellVars{}); err == nil {
		t.Error("an option with no value rendered as an empty string")
	}
}

func TestAShellIdNamesItsPackageAndIsWrittenOnce(t *testing.T) {
	refusedPg(t, strings.Replace(pg, "station:pg/path", "station:other/path", 1), "names another package than pg")
	refusedPg(t, strings.Replace(pg, "station:pg/path", "pg/path", 1), "is `station:<package>` or `station:<package>/<sub>`")
	refusedPg(t, strings.Replace(pg, "station:pg/path", "station:pg/Path", 1), "is `station:<package>` or `station:<package>/<sub>`")
	twice := pg + "          - id: station:pg/path\n            zsh: echo hi\n"
	refusedPg(t, twice, "is written twice")
}

func TestAShellBlockHoldsNoMarkerLine(t *testing.T) {
	refusedPg(t, strings.Replace(pg, "export PGPORT", "# <<< station:pg/path <<<\n                export PGPORT", 1), "holds a marker line")
}

func TestASettingsShellIsItsEffectAndTakesNoOptions(t *testing.T) {
	setting := "package: station.package/v1\nkind: setting\nname: hello\ndescription: d\nmethods:\n    - os: any\n      shell:\n          - id: station:hello\n            zsh: echo {{ home }}\n"
	if _, err := catalog.Decode("packages/shell/hello.yaml", []byte(setting)); err != nil {
		t.Fatal(err)
	}
	refusedAt(t, "packages/shell/hello.yaml", strings.Replace(setting, "{{ home }}", "{{ option.x }}", 1), "a setting's block takes the machine's variables")
	refusedAt(t, "packages/shell/hello.yaml", strings.Replace(setting, "      shell:", "      builtin: dns\n      shell:", 1), "names 2 managers")
}

func TestAnAppsShellNeedsItsManager(t *testing.T) {
	refusedPg(t, strings.Replace(pg, "      brew: pg\n", "", 1), "is no method of a app")
}
