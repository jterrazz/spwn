package main

import (
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	wordsPath      = "packages/api/words.json"
	wordsTypesPath = "packages/api/words.d.ts"
)

// value: what a client draws for one value of a closed enum: its word, a
// card's second line, the singular and plural noun it reads as after a
// count, and its tone.
type word struct {
	Word   string
	Detail string `json:",omitempty"`
	One    string `json:",omitempty"`
	Other  string `json:",omitempty"`
	Tone   verdict.Tone
}

// value: the words of each closed enum a client draws, by the enum's
// TypeScript name, then by value.
type wordTable map[string]map[string]word

type worded struct {
	of    enumerated
	words map[string]word
}

func plain(w string) word { return word{Word: w, Tone: verdict.Plain} }

// invariant: every value of every enum here has its word, and no word names
// a value its enum does not have; words_test.go holds both.
func wordedEnums() []worded {
	return []worded{
		{machinefile.Layer(""), map[string]word{
			string(machinefile.Overlay):     plain("this machine's own changes"),
			string(machinefile.User):        plain("~/.config/station"),
			string(machinefile.Shared):      plain("a shared source"),
			string(machinefile.Builtin):     plain("built into station"),
			string(machinefile.ThisMachine): plain("this machine"),
			string(machinefile.Catalog):     plain("the catalog"),
			string(machinefile.Locked):      plain("the lock"),
		}},
		{machines.Lifetime(""), map[string]word{
			string(machines.Ephemeral):  {Word: machines.Ephemeral.Word(), Detail: "Deleted when it powers off", Tone: verdict.Plain},
			string(machines.Persistent): {Word: machines.Persistent.Word(), Detail: "Stays until you delete it", Tone: verdict.Plain},
			string(machines.AlwaysOn):   plain(machines.AlwaysOn.Word()),
		}},
		{machines.Preset(""), map[string]word{
			string(machines.PresetSmall):     plain(machines.PresetSmall.Word()),
			string(machines.PresetWorkspace): plain(machines.PresetWorkspace.Word()),
			string(machines.PresetGate):      plain(machines.PresetGate.Word()),
		}},
		{machines.Fit(""), map[string]word{
			string(machines.FitsIn): plain("fits"),
			string(machines.NoRoom): plain("no room"),
		}},
		{machines.RoomFor(""), map[string]word{
			string(machines.RoomForGate):      plain("room for one more " + machines.PresetGate.Word()),
			string(machines.RoomForWorkspace): plain("room for one more " + machines.PresetWorkspace.Word()),
			string(machines.RoomForSmall):     plain("room for a " + machines.PresetSmall.Word() + " machine"),
			string(machines.RoomForNone):      plain("no room for another machine"),
		}},
		{machines.BirthStep(""), map[string]word{
			string(machines.BornBuilding): {Word: machines.BornBuilding.Word(), Detail: machines.BornBuilding.Noun(), Tone: verdict.New},
			string(machines.BornCloning):  {Word: machines.BornCloning.Word(), Detail: machines.BornCloning.Noun(), Tone: verdict.New},
			string(machines.BornBooting):  {Word: machines.BornBooting.Word(), Detail: machines.BornBooting.Noun(), Tone: verdict.New},
			string(machines.BornAddress):  {Word: machines.BornAddress.Word(), Detail: machines.BornAddress.Noun(), Tone: verdict.New},
			string(machines.BornSSH):      {Word: machines.BornSSH.Word(), Detail: machines.BornSSH.Noun(), Tone: verdict.New},
		}},
		{machinefile.Chapter(""), map[string]word{
			string(machinefile.ChapterApps):     {Word: "Apps", One: "app", Other: "apps", Tone: verdict.Plain},
			string(machinefile.ChapterSettings): {Word: "Settings", One: "setting", Other: "settings", Tone: verdict.Plain},
			string(machinefile.ChapterPorts):    {Word: "Ports", One: "port", Other: "ports", Tone: verdict.Plain},
			string(machinefile.ChapterAccounts): {Word: "Accounts", One: "account", Other: "accounts", Tone: verdict.Plain},
		}},
		{catalog.OS(""), map[string]word{
			string(catalog.OSMacOS):  plain("macOS"),
			string(catalog.OSUbuntu): plain("Ubuntu"),
			string(catalog.OSDebian): plain("Debian"),
			string(catalog.OSAny):    plain("any system"),
		}},
		{catalog.MachineKind(""), map[string]word{
			string(catalog.Mac):   plain("Mac"),
			string(catalog.Linux): plain("Linux"),
			string(catalog.MacOS): plain("macOS VM"),
		}},
		{catalog.Arch(""), map[string]word{
			string(catalog.ArchARM64): plain(string(catalog.ArchARM64)),
			string(catalog.ArchAMD64): plain(string(catalog.ArchAMD64)),
		}},
	}
}

func wordsName(e enumerated) string { return tsName(qualified(reflect.TypeOf(e))) }

func wordsTable() wordTable {
	out := wordTable{}
	for _, w := range wordedEnums() {
		out[wordsName(w.of)] = w.words
	}
	return out
}

func toneName() string { return tsName(qualified(reflect.TypeFor[verdict.Tone]())) }

// value: the TypeScript names the words declare: each worded enum's, the
// tone's, and the two shapes.
func wordsNames() []string {
	enums := wordedEnums()
	names := make([]string, 0, 3+len(enums))
	names = append(names, toneName(), "StationWord", "StationWords")
	for _, w := range enums {
		names = append(names, wordsName(w.of))
	}
	return names
}

// value: the TypeScript of the words: each worded enum and the tone as a
// union of their values, a word's shape, and StationWords, a Record per
// enum, so a client's compiler refuses a value without its word.
func wordsTypes() string {
	unions := map[string][]string{toneName(): verdict.Tone("").Enum()}
	enums := wordedEnums()
	rows := make([]string, 0, len(enums))
	for _, w := range enums {
		name := wordsName(w.of)
		unions[name] = w.of.Enum()
		rows = append(rows, "    "+name+": Record<"+name+", StationWord>;\n")
	}
	var b strings.Builder
	for _, name := range slices.Sorted(maps.Keys(unions)) {
		parts := make([]string, 0, len(unions[name]))
		for _, v := range unions[name] {
			parts = append(parts, literal(v))
		}
		b.WriteString("\nexport type " + name + " = " + union(parts) + ";\n")
	}
	b.WriteString("\nexport type StationWord = {\n    Detail?: string;\n    One?: string;\n    Other?: string;\n    Tone: " + toneName() + ";\n    Word: string;\n};\n")
	slices.Sort(rows)
	b.WriteString("\nexport type StationWords = {\n" + strings.Join(rows, "") + "};\n")
	return b.String()
}

// value: the type of words.json, so a client's import of it reads each
// enum's words by its own values and a tone as one of the tones, never a
// bare string.
func wordsDeclaration() []byte {
	return []byte("// Generated by `make generate` (apps/engine/cmd/generate) from the closed enums' words: do not edit.\n" +
		"import type { StationWords } from './index.js';\n\ndeclare const words: StationWords;\nexport default words;\n")
}
