package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

type FileError struct {
	File    string
	Line    int
	Problem string
}

func (e *FileError) Error() string {
	if e.Line > 0 {
		return e.File + ":" + strconv.Itoa(e.Line) + ": " + e.Problem
	}
	return e.File + ": " + e.Problem
}

type NewerSchemaError struct {
	File, Schema string
	// value: the version of that format this station reads.
	Reads string
}

func (e *NewerSchemaError) Error() string {
	return e.File + " is " + e.Schema + ", newer than this station reads (" + e.Reads + "): update station"
}

// invariant: YAML 1.1 reads these plain scalars as booleans and octals where
// YAML 1.2 reads strings and decimals; a file says which it means, so either
// reader gets one value.
var (
	yaml11Bool  = regexp.MustCompile(`^(?:y|Y|yes|Yes|YES|n|N|no|No|NO|on|On|ON|off|Off|OFF)$`)
	yaml11Octal = regexp.MustCompile(`^[-+]?0[0-9_]+$`)
	yaml11Sexa  = regexp.MustCompile(`^[-+]?\d[0-9_]*(?::[0-5]?\d)+(?:\.[0-9_]*)?$`)
)

// value: a file format station reads strictly: the key its first line names
// the schema by, the schema this station reads, and the family its versions
// share.
type format struct {
	key, schema, family string
}

var (
	items        = format{key: "item", schema: Schema, family: SchemaFamily}
	machinefiles = format{key: "machinefile", schema: MachinefileSchema, family: MachinefileFamily}
)

func Decode(file string, body []byte) (Item, error) {
	var it Item
	doc, err := strict(file, body, items, &it)
	if err != nil {
		return Item{}, err
	}
	if err := checkItem(file, doc, it); err != nil {
		return Item{}, err
	}
	return it, nil
}

// invariant: one strict read for every format: the schema is judged first,
// so a file of a newer one says `update station` before any refusal of what
// this station reads; then one document, no anchor, alias, merge key nor
// YAML 1.1 scalar, and every key known.
func strict(file string, body []byte, f format, v any) (*yaml.Node, error) {
	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, &FileError{File: file, Problem: "is empty: its first key is `" + f.key + ": " + f.schema + "`"}
		}
		return nil, &FileError{File: file, Problem: yamlProblem(err)}
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil, &FileError{File: file, Line: doc.Line, Problem: "is not a mapping of keys"}
	}
	if err := schemaOf(file, doc, f); err != nil {
		return nil, err
	}
	var more yaml.Node
	if err := dec.Decode(&more); !errors.Is(err, io.EOF) {
		return nil, &FileError{File: file, Problem: "holds more than one document"}
	}
	if err := plainYAML(file, &root); err != nil {
		return nil, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	decoder.KnownFields(true)
	if err := decoder.Decode(v); err != nil {
		return nil, &FileError{File: file, Problem: yamlProblem(err)}
	}
	return doc, nil
}

func yamlProblem(err error) string {
	return strings.ReplaceAll(strings.TrimPrefix(err.Error(), "yaml: "), "\n ", "")
}

func plainYAML(file string, n *yaml.Node) error {
	if n.Anchor != "" {
		return &FileError{File: file, Line: n.Line, Problem: "an anchor (&" + n.Anchor + "): write the value out"}
	}
	switch n.Kind {
	case yaml.AliasNode:
		return &FileError{File: file, Line: n.Line, Problem: "an alias (*" + n.Value + "): write the value out"}
	case yaml.ScalarNode:
		if n.Tag == "!!merge" {
			return &FileError{File: file, Line: n.Line, Problem: "a merge key (<<): write the keys out"}
		}
		if n.Style == 0 && (yaml11Bool.MatchString(n.Value) || yaml11Octal.MatchString(n.Value) || yaml11Sexa.MatchString(n.Value)) {
			return &FileError{File: file, Line: n.Line, Problem: fmt.Sprintf("%q reads differently in YAML 1.1: write true or false, 0o for an octal, or quote it", n.Value)}
		}
	case yaml.DocumentNode, yaml.SequenceNode, yaml.MappingNode:
	}
	for _, c := range n.Content {
		if err := plainYAML(file, c); err != nil {
			return err
		}
	}
	return nil
}

func schemaOf(file string, doc *yaml.Node, f format) error {
	key, value := entry(doc, f.key)
	if value == nil {
		return &FileError{File: file, Line: doc.Line, Problem: "names no schema: its first key is `" + f.key + ": " + f.schema + "`"}
	}
	if value.Value == f.schema {
		return nil
	}
	if regexp.MustCompile(`^` + regexp.QuoteMeta(f.family) + `/v([1-9][0-9]*)$`).MatchString(value.Value) {
		return &NewerSchemaError{File: file, Schema: value.Value, Reads: f.schema}
	}
	return &FileError{File: file, Line: key.Line, Problem: fmt.Sprintf("names the schema %q, not %s", value.Value, f.schema)}
}

func entry(doc *yaml.Node, key string) (k, v *yaml.Node) {
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == key {
			return doc.Content[i], doc.Content[i+1]
		}
	}
	return nil, nil
}

type enumerated interface{ Enum() []string }

func known[E interface {
	enumerated
	~string
}](v E) bool {
	return slices.Contains(v.Enum(), string(v))
}

// invariant: one check per key, in the file's order, each saying the key it
// judges: a refusal names the line the owner fixes.
func checkItem(file string, doc *yaml.Node, it Item) error {
	if !known(it.Kind) {
		return refused(file, doc, "kind", fmt.Sprintf("kind %q is none of %s", it.Kind, strings.Join(it.Kind.Enum(), ", ")))
	}
	if want := strings.TrimSuffix(path.Base(file), ".yaml"); it.Name != want {
		return refused(file, doc, "name", fmt.Sprintf("names %q, its file %q", it.Name, want))
	}
	if it.Description == "" {
		return refused(file, doc, "description", "has no description")
	}
	if key := foreignKey(it); key != "" {
		return refused(file, doc, key, "`"+key+"` is not a key of a "+string(it.Kind))
	}
	if it.Kind == KindSetting && len(it.Requires) > 1 {
		return refused(file, doc, "requires", "a setting requires one app; name alternatives in requires-one-of")
	}
	if len(it.Requires) > 0 && len(it.RequiresOneOf) > 0 {
		return refused(file, doc, "requires-one-of", "requires and requires-one-of, one of them")
	}
	if it.Deprecated != nil && it.Deprecated.Why == "" {
		return refused(file, doc, "deprecated", "deprecated says why")
	}
	if problem := checkManages(it); problem != "" {
		return refused(file, doc, "manages", problem)
	}
	for _, p := range it.Ports {
		if p.Port < 1 || p.Port > 65535 || p.Label == "" {
			return refused(file, doc, "ports", fmt.Sprintf("port %d: a port from 1 to 65535, and its label", p.Port))
		}
		if problem := checkPortEnv(p); problem != "" {
			return refused(file, doc, "ports", problem)
		}
	}
	if it.Phase != "" && !known(it.Phase) {
		return refused(file, doc, "phase", fmt.Sprintf("phase %q is none of %s", it.Phase, strings.Join(it.Phase.Enum(), ", ")))
	}
	if slices.Contains(it.RenamedFrom, it.Name) {
		return refused(file, doc, "renamed-from", "an item is not renamed from its own name")
	}
	for _, a := range it.Accepts {
		if !known(a) {
			return refused(file, doc, "accepts", fmt.Sprintf("source %q is none of %s", a, strings.Join(a.Enum(), ", ")))
		}
	}
	for _, c := range it.Provides {
		if err := CheckCommand(c); err != nil {
			return refused(file, doc, "provides", err.Error())
		}
	}
	for _, n := range it.Needs {
		if !known(n) {
			return refused(file, doc, "needs", fmt.Sprintf("need %q is none of %s", n, strings.Join(n.Enum(), ", ")))
		}
	}
	for _, a := range it.AfterApply {
		if !known(a) {
			return refused(file, doc, "after-apply", fmt.Sprintf("%q is none of %s", a, strings.Join(a.Enum(), ", ")))
		}
	}
	for _, o := range it.Options {
		if problem := checkOption(o); problem != "" {
			return refused(file, doc, "options", "option "+o.Name+": "+problem)
		}
	}
	if len(it.Methods) == 0 {
		return refused(file, doc, "methods", "has no method")
	}
	for i, m := range it.Methods {
		if problem := checkMethod(it, m); problem != "" {
			return refused(file, doc, "methods", fmt.Sprintf("method %d: %s", i+1, problem))
		}
		if problem := references(reflect.ValueOf(m), it.Options); problem != "" {
			return refused(file, doc, "methods", fmt.Sprintf("method %d: %s", i+1, problem))
		}
	}
	if problem := checkStored(it.Writes); problem != "" {
		return refused(file, doc, "writes", problem)
	}
	if it.Disk != nil {
		if problem := checkDisk(*it.Disk); problem != "" {
			return refused(file, doc, "disk", problem)
		}
	}
	return nil
}

func refused(file string, doc *yaml.Node, key, problem string) error {
	line := doc.Line
	if k, _ := entry(doc, key); k != nil {
		line = k.Line
	}
	return &FileError{File: file, Line: line, Problem: problem}
}

// invariant: a key belongs to the kinds that read it: an app's disk row is
// no key of a setting, a setting's writes none of an app's.
func foreignKey(it Item) string {
	type key struct {
		name string
		set  bool
	}
	app := []key{
		{"replaces", it.Replaces != ""}, {"provides", len(it.Provides) > 0}, {"then", len(it.Then) > 0},
		{"bundle", it.Bundle != ""}, {"version-read", it.Version != nil}, {"supersedes", it.Supersedes != ""},
		{"accepts", len(it.Accepts) > 0}, {"updates", it.Updates != ""}, {"live", it.Live != nil}, {"disk", it.Disk != nil},
		{"ports", len(it.Ports) > 0},
	}
	setting := []key{
		{"help", it.Help != ""}, {"requires-one-of", len(it.RequiresOneOf) > 0},
		{"guided", it.Guided}, {"reversible", it.Reversible}, {"run-once", it.RunOnce},
		{"declares", it.Declares != nil}, {"sets", len(it.Sets) > 0}, {"writes", len(it.Writes) > 0},
	}
	foreign := setting
	if it.Kind == KindSetting {
		foreign = app
	}
	if it.Kind != KindManager {
		foreign = append(foreign, key{"manages", len(it.Manages) > 0}, key{"priority", it.Priority != nil})
	}
	for _, k := range foreign {
		if k.set {
			return k.name
		}
	}
	return ""
}

func (m MethodSpec) appKeys() []string {
	var keys []string
	for key, set := range map[string]bool{
		"brew": m.Brew != "", "cask": m.Cask != "", "mas": m.Mas != nil, "apt": m.Apt != nil, "npm": m.Npm != "", "uv": m.Uv != "",
		"bun": m.Bun != "", "mise": m.Mise != nil, "github-release": m.GitHubRelease != nil, "script": m.Script != nil,
		"skill": m.Skill != nil, "provided-by": m.ProvidedBy != "",
	} {
		if set {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func (m MethodSpec) settingKeys() []string {
	var keys []string
	for key, set := range map[string]bool{
		"file": m.File != nil, "builtin": m.Builtin != "", "macos-defaults": m.MacOSDefaults != nil, "git-aliases": m.GitAliases != nil,
		"zsh-functions": m.ZshFunctions != nil, "agent-hooks": m.AgentHooks != nil, "mux-machines": m.MuxMachines != nil,
	} {
		if set {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// value: a version range: comparisons joined by commas, `>=24.04`,
// `>=14, <27`, `15`.
var versionRange = regexp.MustCompile(`^(?:>=|<=|>|<|=)?\d+(?:\.\d+)*(?:, ?(?:>=|<=|>|<|=)?\d+(?:\.\d+)*)*$`)

// invariant: a method names its system by `os` or `family`, one of them; a
// disk row may name none (every machine). A version range is a version of
// one os.
func checkSelector(s Selector, method bool) string {
	switch {
	case s.OS != "" && s.Family != "":
		return "names an os and a family: one of them"
	case method && s.OS == "" && s.Family == "":
		return "names no os nor family (`os: any` is every system)"
	case s.OS != "" && !known(s.OS):
		return fmt.Sprintf("os %q is none of %s", s.OS, strings.Join(s.OS.Enum(), ", "))
	case s.Family != "" && !known(s.Family):
		return fmt.Sprintf("family %q is none of %s", s.Family, strings.Join(s.Family.Enum(), ", "))
	case s.Arch != "" && !known(s.Arch):
		return fmt.Sprintf("arch %q is none of %s", s.Arch, strings.Join(s.Arch.Enum(), ", "))
	case s.Version != "" && (s.OS == "" || s.OS == OSAny):
		return "a version range is of one os"
	case s.Version != "" && !versionRange.MatchString(s.Version):
		return fmt.Sprintf("version %q is no range (`>=24.04`, `>=14, <27`)", s.Version)
	}
	for _, k := range s.Machine {
		if !known(k) {
			return fmt.Sprintf("kind %q is none of %s", k, strings.Join(k.Enum(), ", "))
		}
	}
	return ""
}

func checkMethod(it Item, m MethodSpec) string {
	if problem := checkSelector(m.Selector, true); problem != "" {
		return problem
	}
	own, other := m.appKeys(), m.settingKeys()
	if it.Kind == KindSetting {
		own, other = other, own
	}
	if len(other) > 0 {
		return "`" + other[0] + "` is no method of a " + string(it.Kind)
	}
	if len(own) != 1 {
		return fmt.Sprintf("names %d managers (%s), not one", len(own), strings.Join(own, ", "))
	}
	if problem := m.checkOverrides(it.Kind); problem != "" {
		return problem
	}
	if problem := m.checkManager(); problem != "" {
		return problem
	}
	if it.Kind == KindSetting {
		return ""
	}
	if m.ProvidedBy != "" && len(it.Provides) == 0 && len(m.Provides) == 0 {
		return "an app " + m.ProvidedBy + " provides names its commands in provides"
	}
	if m.GitHubRelease != nil && len(it.Provides) == 0 && len(m.Provides) == 0 {
		return "a release lands the commands its app names in provides"
	}
	if m.Apt != nil && m.Family != FamilyDebian && m.OS != OSUbuntu && m.OS != OSDebian {
		return "apt runs on the debian family"
	}
	if m.Apt != nil && !slices.Contains(m.Needs, NeedSudo) {
		return "apt installs as root: its method says `needs: [sudo]`"
	}
	if install := m.install(nil); install != nil {
		for _, k := range m.kinds() {
			if !slices.Contains(MethodKinds[install.Method()], k) {
				return fmt.Sprintf("%s does not run on %s", install.Method(), k)
			}
		}
	}
	return ""
}

// invariant: the keys that change what a method's manager does by default
// are an app's, and each its own manager's where it says so.
func (m MethodSpec) checkOverrides(kind Kind) string {
	switch {
	case m.Trusted && m.Brew == "" && m.Cask == "":
		return "`trusted` is for a brew or a cask"
	case m.Package != "" && m.Bun == "":
		return "`package` is for a bun install"
	case kind == KindSetting && (len(m.Path) > 0 || len(m.Provides) > 0 || m.Upgrade != "" || m.Pin != ""):
		return "`path`, `provides`, `upgrade` and `pin` are for an app's method"
	case m.Pin != "" && !known(m.Pin):
		return fmt.Sprintf("pin %q is none of %s", m.Pin, strings.Join(m.Pin.Enum(), ", "))
	}
	for _, p := range m.Path {
		if !strings.HasPrefix(p, "/") {
			return "path " + p + " is not absolute"
		}
	}
	for _, c := range m.Provides {
		if err := CheckCommand(c); err != nil {
			return err.Error()
		}
	}
	for _, n := range m.Needs {
		if !known(n) {
			return fmt.Sprintf("need %q is none of %s", n, strings.Join(n.Enum(), ", "))
		}
	}
	return ""
}

func (m MethodSpec) checkManager() string {
	switch {
	case m.Mas != nil && (m.Mas.ID <= 0 || m.Mas.Name == ""):
		return "mas names the App Store id and name"
	case m.Mise != nil && (m.Mise.Name == "" || m.Mise.Version == ""):
		return "mise names what it installs and its version"
	case m.Skill != nil && (m.Skill.Repo == "" || m.Skill.Name == ""):
		return "skill names a repo and the skill's name"
	case m.GitHubRelease != nil && (m.GitHubRelease.Repo == "" || m.GitHubRelease.Asset == ""):
		return "github-release names its repo (`owner/name`) and its asset's pattern"
	case m.Script != nil:
		return m.Script.check(m.Upgrade)
	case m.File != nil && (m.File.Touch == "") == (len(m.File.Copy) == 0):
		return "file touches one path or copies files, one of them"
	case m.File != nil && m.File.Touch != "" && len(m.File.Reload) > 0:
		return "a touched file reloads nothing"
	case m.Apt != nil && len(m.Apt.Packages) == 0:
		return "apt names its packages"
	case m.File != nil && m.File.Mode != "" && !known(m.File.Mode):
		return fmt.Sprintf("file's mode %q is none of %s", m.File.Mode, strings.Join(m.File.Mode.Enum(), ", "))
	case m.MacOSDefaults != nil:
		return m.MacOSDefaults.check(m.Selector)
	}
	if m.GitHubRelease != nil {
		if err := CheckRepo(m.GitHubRelease.Repo); err != nil {
			return err.Error()
		}
	}
	if m.Skill != nil {
		if err := CheckRepo(m.Skill.Repo); err != nil {
			return err.Error()
		}
		if err := CheckSkill(m.Skill.Name); err != nil {
			return err.Error()
		}
	}
	if m.Apt != nil {
		for _, p := range m.Apt.Packages {
			if err := CheckPackage(p); err != nil {
				return err.Error()
			}
		}
	}
	return ""
}

func (s ScriptMethod) check(upgrade string) string {
	switch {
	case s.Run != "" && s.URL != "":
		return "script installs by a run line or a url, one of them"
	case s.Run == "" && s.URL == "" && s.Hint == "":
		return "script installs by a run line or a url, or says the owner's way in its hint"
	case s.Run == "" && s.URL == "" && (s.Check != "" || s.Undo != "" || upgrade != ""):
		return "a script the owner installs has a hint, no check, undo or upgrade"
	case s.Run != "" && (len(s.Args) > 0 || len(s.Env) > 0 || upgrade != ""):
		return "a run line takes no args, env or upgrade"
	}
	return ""
}

func (d MacOSDefaults) check(s Selector) string {
	if s.OS != OSMacOS {
		return "macos-defaults is for os: macos"
	}
	if d.Domain == "" || len(d.Keys) == 0 {
		return "macos-defaults names its domain and its keys"
	}
	for _, k := range d.Keys {
		switch {
		case !known(k.Type):
			return fmt.Sprintf("key %s's type %q is none of %s", k.Key, k.Type, strings.Join(k.Type.Enum(), ", "))
		case k.Key == "":
			return "a key of macos-defaults names its key"
		}
	}
	return ""
}

// invariant: an option's keys are its type's: values for an enum, bounds
// for an integer, secret for a string; its default and examples are values
// of its type.
func checkOption(o Option) string {
	switch {
	case o.Name == "":
		return "has no name"
	case !known(o.Type):
		return fmt.Sprintf("type %q is none of %s", o.Type, strings.Join(o.Type.Enum(), ", "))
	case (o.Type == OptionEnum) != (len(o.Values) > 0):
		return "an enum lists its values, and only an enum"
	case (o.Min != nil || o.Max != nil) && o.Type != OptionInteger:
		return "min and max bound an integer"
	case o.Min != nil && o.Max != nil && *o.Min > *o.Max:
		return "min is above max"
	case o.Secret && o.Type != OptionString:
		return "a secret is a string"
	case o.Type == OptionBoolean && len(o.Examples) > 0:
		return "a boolean has a default, no examples"
	}
	values := slices.Clone(o.Examples)
	if o.Default != nil {
		values = append(values, *o.Default)
	}
	for _, v := range values {
		if problem := o.fits(string(v)); problem != "" {
			return problem
		}
	}
	return ""
}

func (o Option) fits(v string) string {
	switch o.Type {
	case OptionInteger:
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Sprintf("%q is no integer", v)
		}
		if o.Min != nil && n < *o.Min || o.Max != nil && n > *o.Max {
			return fmt.Sprintf("%d is out of its bounds", n)
		}
	case OptionBoolean:
		if v != "true" && v != "false" {
			return fmt.Sprintf("%q is no boolean", v)
		}
	case OptionEnum:
		if !slices.Contains(o.Values, v) {
			return fmt.Sprintf("%q is none of its values", v)
		}
	case OptionString:
	}
	return ""
}

func checkStored(paths []Stored) string {
	for _, p := range paths {
		if !known(p.Level) {
			return fmt.Sprintf("%s's level %q is none of %s", p.Path, p.Level, strings.Join(p.Level.Enum(), ", "))
		}
		if problem := checkSelector(p.Placed, false); problem != "" {
			return p.Path + ": " + problem
		}
	}
	return ""
}

func checkDisk(d Disk) string {
	if problem := checkStored(d.Paths); problem != "" {
		return problem
	}
	if d.Size != SizeDirs && !known(d.Size) {
		return fmt.Sprintf("size %q is none of %s", d.Size, strings.Join(d.Size.Enum(), ", "))
	}
	actions := slices.Concat(d.Free, d.Erase)
	for _, m := range d.Misplaced {
		actions = append(actions, m.Free...)
	}
	for _, a := range actions {
		if problem := checkSelector(a.Placed, false); problem != "" {
			return problem
		}
	}
	for _, r := range d.Relocate {
		if !known(r.File) {
			return fmt.Sprintf("relocate's file %q is none of %s", r.File, strings.Join(r.File.Enum(), ", "))
		}
	}
	return ""
}

// invariant: a manager names the method keys it installs by, and only a
// manager does; a priority is a rank, from 1.
func checkManages(it Item) string {
	for _, k := range it.Manages {
		if !known(k) {
			return fmt.Sprintf("%q is none of %s", k, strings.Join(k.Enum(), ", "))
		}
	}
	if p := it.Priority; p != nil {
		if p.MacOS < 0 || p.Ubuntu < 0 || p.Debian < 0 {
			return "a priority is a rank, from 1"
		}
		if *p == (Priority{}) {
			return "a priority ranks at least one system"
		}
		if len(it.Manages) == 0 {
			return "a manager with a priority names what it manages"
		}
	}
	return ""
}

// value: the placeholders a port's env template may use.
var portPlaceholders = []string{"{url}", "{host}"}

var portPlaceholder = regexp.MustCompile(`\{[^}]*\}`)

// invariant: a port's env value is a template whose only braces are `{url}`
// and `{host}`; a name is a variable name, never empty.
func checkPortEnv(p PortHint) string {
	for _, name := range slices.Sorted(maps.Keys(p.Env)) {
		if name == "" || strings.ContainsAny(name, "= ") {
			return fmt.Sprintf("port %d: env name %q is no variable name", p.Port, name)
		}
		for _, b := range portPlaceholder.FindAllString(p.Env[name], -1) {
			if !slices.Contains(portPlaceholders, b) {
				return fmt.Sprintf("port %d: env %s uses %s, only %s are known", p.Port, name, b, strings.Join(portPlaceholders, " and "))
			}
		}
	}
	return ""
}

// value: a reference inside a value: `${option.<name>}` is an option's
// value. The braces without a dollar (`{cache}`, `{os}`) are station's own
// placeholders, and a shell's `${HOME}` stays the shell's.
var reference = regexp.MustCompile(`\$\{option\.([^}]*)\}`)

// invariant: every `${option.<name>}` inside a method's values names an
// option of the item; one that names none is refused at decode.
func references(v reflect.Value, options []Option) string {
	switch v.Kind() {
	case reflect.String:
		for _, m := range reference.FindAllStringSubmatch(v.String(), -1) {
			if !hasOption(options, m[1]) {
				return m[0] + " names no option of the item"
			}
		}
	case reflect.Pointer:
		if !v.IsNil() {
			return references(v.Elem(), options)
		}
	case reflect.Slice:
		for i := range v.Len() {
			if problem := references(v.Index(i), options); problem != "" {
				return problem
			}
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if problem := references(v.Field(i), options); problem != "" {
				return problem
			}
		}
	default:
	}
	return ""
}

func hasOption(options []Option, name string) bool {
	for _, o := range options {
		if o.Name == name {
			return true
		}
	}
	return false
}
