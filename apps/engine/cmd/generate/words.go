package main

import (
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

const (
	wordsPath      = "packages/api/words.json"
	wordsTypesPath = "packages/api/words.d.ts"
)

// value: what a client draws for one value of a closed enum: its word, a
// second wording (a card's second line, a shorter name of the same value),
// what a list of the lines it names is headed, the singular and plural noun
// it reads as after a count, and its tone.
type word struct {
	Word    string
	Detail  string `json:",omitempty"`
	Heading string `json:",omitempty"`
	One     string `json:",omitempty"`
	Other   string `json:",omitempty"`
	Tone    verdict.Tone
}

// value: the words of each closed enum a client draws, by the enum's
// TypeScript name, then by value.
type wordTable map[string]map[string]word

type worded struct {
	of    enumerated
	words map[string]word
}

func plain(w string) word { return word{Word: w, Tone: verdict.Plain} }

func toned(t verdict.Tone, w string) word { return word{Word: w, Tone: t} }

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
		{machinefile.Condition(""), map[string]word{
			string(machinefile.Present):    toned(verdict.OK, "present"),
			string(machinefile.Absent):     toned(verdict.Warn, "not installed"),
			string(machinefile.Outdated):   toned(verdict.Warn, "outdated"),
			string(machinefile.Changed):    toned(verdict.Warn, "changed"),
			string(machinefile.NotManaged): plain("not managed"),
			string(machinefile.Extra):      plain("extra"),
			string(machinefile.Unknown):    plain("not known"),
		}},
		{machinefile.Action(""), map[string]word{
			string(machinefile.Install): {Word: "install it", Heading: "To install", Tone: verdict.New},
			string(machinefile.Set):     {Word: "set it", Heading: "To set", Tone: verdict.New},
			string(machinefile.Adopt):   {Word: "adopt it", Heading: "To adopt", Tone: verdict.New},
			string(machinefile.Keep):    {Word: "keep it", Heading: "Unchanged", Tone: verdict.New},
			string(machinefile.Leave):   {Word: "leave it", Heading: "Left as is", Tone: verdict.New},
			string(machinefile.Remove):  {Word: "remove it", Heading: "To remove", Tone: verdict.New},
		}},
		{catalog.Level(""), map[string]word{
			string(catalog.LevelCache):    {Word: "cache", Detail: "rebuilt when needed", Tone: verdict.Plain},
			string(catalog.LevelData):     {Word: "data", Detail: "yours, kept", Tone: verdict.Plain},
			string(catalog.LevelSettings): {Word: "settings", Detail: "your settings, kept", Tone: verdict.Plain},
			string(catalog.LevelKept):     {Word: "kept", Detail: "kept", Tone: verdict.Plain},
		}},
		{catalog.Audience(""), map[string]word{
			string(catalog.AudienceLocal):   plain("Local"),
			string(catalog.AudiencePrivate): plain("Private"),
			string(catalog.AudiencePublic):  plain("Public"),
		}},
		{catalog.Need(""), map[string]word{
			string(catalog.NeedSudo):          plain("your password"),
			string(catalog.NeedTerminal):      plain("a terminal"),
			string(catalog.NeedLicense):       plain("a license accepted"),
			string(catalog.NeedAppStoreLogin): plain("an App Store sign-in"),
		}},
		{catalog.AfterApply(""), map[string]word{
			string(catalog.AfterLogout):    plain("a log out"),
			string(catalog.AfterReboot):    plain("a restart of the machine"),
			string(catalog.RestartDock):    plain("the Dock restarted"),
			string(catalog.RestartFinder):  plain("the Finder restarted"),
			string(catalog.RestartMenuBar): plain("the menu bar restarted"),
		}},
		{catalog.Kind(""), map[string]word{
			string(catalog.KindApp):     plain("app"),
			string(catalog.KindManager): plain("manager"),
			string(catalog.KindSetting): plain("setting"),
		}},
		{disk.Group(""), map[string]word{
			string(disk.Misplaced):    {Word: "Misplaced", Detail: "misplaced", Tone: verdict.Plain},
			string(disk.CanRebuild):   {Word: "Can rebuild", Detail: "cache", Tone: verdict.Plain},
			string(disk.YourData):     {Word: "Your data", Detail: "data", Tone: verdict.Plain},
			string(disk.NeverTouched): {Word: "Never touched", Detail: "never touched", Tone: verdict.Plain},
		}},
		{machines.VMState(""), map[string]word{
			string(machines.Running):      toned(verdict.OK, machines.Running.Word()),
			string(machines.Stopped):      plain(machines.Stopped.Word()),
			string(machines.Suspended):    plain(machines.Suspended.Word()),
			string(machines.Stale):        toned(verdict.Warn, machines.Stale.Word()),
			string(machines.Broken):       toned(verdict.Bad, machines.Broken.Word()),
			string(machines.Creating):     toned(verdict.New, machines.Creating.Word()),
			string(machines.UnknownState): plain(machines.UnknownState.Word()),
		}},
		{machines.CheckState(""), map[string]word{
			string(machines.CheckFailing): toned(verdict.Bad, "fails"),
			string(machines.CheckPassing): toned(verdict.OK, "passes"),
			string(machines.CheckWaiting): toned(verdict.New, "waiting"),
		}},
		{engine.ReadState(""), map[string]word{
			string(engine.EngineRunning):      toned(verdict.OK, "running"),
			string(engine.EngineStopped):      plain("stopped"),
			string(engine.EngineNotInstalled): plain("not installed"),
		}},
		{fail.Type(""), map[string]word{
			string(fail.Refused):           toned(verdict.Bad, "refused"),
			string(fail.NotFound):          plain("not found"),
			string(fail.NotForThisMachine): plain("not for this machine"),
			string(fail.NeedsConfirmation): toned(verdict.Warn, "needs confirmation"),
			string(fail.NeedsTerminal):     toned(verdict.Warn, "needs a terminal"),
			string(fail.Declined):          plain("declined"),
			string(fail.Unreachable):       toned(verdict.Bad, "unreachable"),
			string(fail.EngineStopped):     toned(verdict.Bad, "engine stopped"),
			string(fail.FarTooOld):         toned(verdict.Bad, "far too old"),
			string(fail.Failed):            toned(verdict.Bad, "failed"),
			string(fail.Interrupted):       toned(verdict.Warn, "interrupted"),
		}},
		{finding.Severity(""), map[string]word{
			string(finding.Broken):     toned(verdict.Bad, "broken"),
			string(finding.NeedsALook): toned(verdict.Warn, "needs a look"),
		}},
		{section.ID(""), sectionWords()},
		{refusalCode{}, refusalWords()},
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

// value: the refusal codes, worded by surface/cli/words, the one owner of
// them: each is called by its name read as words, and carries its family.
type refusalCode struct{}

func (refusalCode) Enum() []string {
	codes := words.CodesSorted()
	out := make([]string, len(codes))
	for i, code := range codes {
		out[i] = string(code)
	}
	return out
}

func (refusalCode) TSName() string { return "CoreFailCode" }

func refusalWords() map[string]word {
	out := map[string]word{}
	for _, code := range words.CodesSorted() {
		name, family := words.Title(code)
		out[string(code)] = word{Word: name, Detail: family, Tone: verdict.Plain}
	}
	return out
}

func sectionWords() map[string]word {
	out := map[string]word{}
	for _, id := range section.Order() {
		out[string(id)] = plain(id.Title())
	}
	return out
}

func wordsName(e enumerated) string {
	if named, ok := e.(interface{ TSName() string }); ok {
		return named.TSName()
	}
	return tsName(qualified(reflect.TypeOf(e)))
}

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
