package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
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
}

func (e *NewerSchemaError) Error() string {
	return e.File + " is " + e.Schema + ", newer than this station reads (" + Schema + "): update station"
}

// invariant: YAML 1.1 reads these plain scalars as booleans and octals where
// YAML 1.2 reads strings and decimals; a file says which it means, so either
// reader gets one value.
var (
	yaml11Bool  = regexp.MustCompile(`^(?:y|Y|yes|Yes|YES|n|N|no|No|NO|on|On|ON|off|Off|OFF)$`)
	yaml11Octal = regexp.MustCompile(`^[-+]?0[0-9_]+$`)
	yaml11Sexa  = regexp.MustCompile(`^[-+]?\d[0-9_]*(?::[0-5]?\d)+(?:\.[0-9_]*)?$`)
	schemaID    = regexp.MustCompile(`^` + regexp.QuoteMeta(SchemaFamily) + `/v([1-9][0-9]*)$`)
)

func Decode(file string, body []byte) (Item, error) {
	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&root); err != nil {
		return Item{}, &FileError{File: file, Problem: yamlProblem(err)}
	}
	var more yaml.Node
	if err := dec.Decode(&more); !errors.Is(err, io.EOF) {
		return Item{}, &FileError{File: file, Problem: "holds more than one document"}
	}
	if err := plainYAML(file, &root); err != nil {
		return Item{}, err
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return Item{}, &FileError{File: file, Line: doc.Line, Problem: "is not a mapping of keys"}
	}
	if err := schemaOf(file, doc); err != nil {
		return Item{}, err
	}
	var it Item
	strict := yaml.NewDecoder(bytes.NewReader(body))
	strict.KnownFields(true)
	if err := strict.Decode(&it); err != nil {
		return Item{}, &FileError{File: file, Problem: yamlProblem(err)}
	}
	if err := checkItem(file, doc, it); err != nil {
		return Item{}, err
	}
	return it, nil
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

func schemaOf(file string, doc *yaml.Node) error {
	key, value := entry(doc, "item")
	if value == nil {
		return &FileError{File: file, Line: doc.Line, Problem: "names no schema: its first key is `item: " + Schema + "`"}
	}
	if value.Value == Schema {
		return nil
	}
	if m := schemaID.FindStringSubmatch(value.Value); m != nil {
		return &NewerSchemaError{File: file, Schema: value.Value}
	}
	return &FileError{File: file, Line: key.Line, Problem: fmt.Sprintf("names the schema %q, not %s", value.Value, Schema)}
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
	if it.Order < 0 {
		return refused(file, doc, "order", "has a negative order")
	}
	if key := foreignKey(it); key != "" {
		return refused(file, doc, key, "`"+key+"` is not a key of a "+string(it.Kind))
	}
	if it.Kind == KindSetting && len(it.Requires) > 1 {
		return refused(file, doc, "requires", "a setting requires one app; name alternatives in requires-one-of")
	}
	for _, n := range it.Needs {
		if !known(n) {
			return refused(file, doc, "needs", fmt.Sprintf("need %q is none of %s", n, strings.Join(n.Enum(), ", ")))
		}
	}
	for _, o := range it.Options {
		if !known(o.Type) {
			return refused(file, doc, "options", fmt.Sprintf("option %s's type %q is none of %s", o.Name, o.Type, strings.Join(o.Type.Enum(), ", ")))
		}
	}
	if len(it.Methods) == 0 {
		return refused(file, doc, "methods", "has no method")
	}
	if it.Kind == KindSetting && len(it.Methods) > 1 {
		return refused(file, doc, "methods", "a setting has one method in "+Schema)
	}
	for i, m := range it.Methods {
		if problem := checkMethod(it.Kind, m); problem != "" {
			return refused(file, doc, "methods", fmt.Sprintf("method %d: %s", i+1, problem))
		}
		if m.ProvidedBy != "" && len(it.Provides) == 0 {
			return refused(file, doc, "methods", fmt.Sprintf("method %d: an app %s provides names its commands in provides", i+1, m.ProvidedBy))
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
// no key of a setting, a setting's options none of an app's.
func foreignKey(it Item) string {
	type key struct {
		name string
		set  bool
	}
	app := []key{
		{"replaces", it.Replaces != ""}, {"provides", len(it.Provides) > 0}, {"then", len(it.Then) > 0},
		{"bundle", it.Bundle != ""}, {"version", it.Version != nil}, {"supersedes", it.Supersedes != ""},
		{"app-store", it.AppStore}, {"live", it.Live != nil}, {"disk", it.Disk != nil},
	}
	setting := []key{
		{"help", it.Help != ""}, {"requires-one-of", len(it.RequiresOneOf) > 0},
		{"guided", it.Guided}, {"reversible", it.Reversible}, {"run-once", it.RunOnce}, {"options", len(it.Options) > 0},
		{"declares", it.Declares != nil}, {"sets", len(it.Sets) > 0}, {"writes", len(it.Writes) > 0},
	}
	foreign := setting
	if it.Kind == KindSetting {
		foreign = app
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
		"brew": m.Brew != "", "cask": m.Cask != "", "mas": m.Mas != nil, "npm": m.Npm != "", "uv": m.Uv != "",
		"bun": m.Bun != "", "mise": m.Mise != nil, "script": m.Script != nil, "skill": m.Skill != nil,
		"provided-by": m.ProvidedBy != "", "manual": m.Manual != "",
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
		"file": m.File != nil, "builtin": m.Builtin != "", "git-aliases": m.GitAliases != nil,
		"zsh-functions": m.ZshFunctions != nil, "agent-hooks": m.AgentHooks != nil, "mux-machines": m.MuxMachines != nil,
	} {
		if set {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func checkMethod(kind Kind, m MethodSpec) string {
	if !known(m.OS) {
		return fmt.Sprintf("os %q is none of %s", m.OS, strings.Join(m.OS.Enum(), ", "))
	}
	if m.Only != "" && !known(m.Only) {
		return fmt.Sprintf("only %q is none of %s", m.Only, strings.Join(m.Only.Enum(), ", "))
	}
	if m.OS == OSLinux && m.Only == OnlyMac {
		return "a Mac runs no linux"
	}
	own, other := m.appKeys(), m.settingKeys()
	if kind == KindSetting {
		own, other = other, own
	}
	if len(other) > 0 {
		return "`" + other[0] + "` is no method of a " + string(kind)
	}
	if len(own) != 1 {
		return fmt.Sprintf("names %d managers (%s), not one", len(own), strings.Join(own, ", "))
	}
	if m.Trusted && m.Brew == "" && m.Cask == "" {
		return "`trusted` is for a brew or a cask"
	}
	if m.Package != "" && m.Bun == "" {
		return "`package` is for a bun install"
	}
	if kind == KindSetting && len(m.Path) > 0 {
		return "`path` is for an app's method"
	}
	for _, p := range m.Path {
		if !strings.HasPrefix(p, "/") {
			return "path " + p + " is not absolute"
		}
	}
	if problem := m.checkManager(); problem != "" {
		return problem
	}
	if kind != KindSetting {
		install := m.install(nil)
		for _, k := range m.kinds() {
			if !slices.Contains(MethodKinds[install.Method()], k) {
				return fmt.Sprintf("%s does not run on %s", install.Method(), k)
			}
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
	case m.Script != nil && (m.Script.URL == "") == (m.Script.Run == ""):
		return "script is a url or a run line, one of them"
	case m.Script != nil && m.Script.Run != "" && (len(m.Script.Args) > 0 || len(m.Script.Env) > 0 || m.Script.Upgrade != ""):
		return "a run line takes no args, env or upgrade"
	case m.File != nil && (m.File.Touch == "") == (len(m.File.Copy) == 0):
		return "file touches one path or copies files, one of them"
	case m.File != nil && m.File.Touch != "" && len(m.File.Reload) > 0:
		return "a touched file reloads nothing"
	case m.Builtin != "" && !known(m.Builtin):
		return fmt.Sprintf("builtin %q is no procedure station knows", m.Builtin)
	}
	return ""
}

func checkStored(paths []Stored) string {
	for _, p := range paths {
		if !known(p.Level) {
			return fmt.Sprintf("%s's level %q is none of %s", p.Path, p.Level, strings.Join(p.Level.Enum(), ", "))
		}
		if problem := checkOn(p.On); problem != "" {
			return problem
		}
	}
	return ""
}

func checkOn(on []MachineKind) string {
	for _, k := range on {
		if !known(k) {
			return fmt.Sprintf("on %q is none of %s", k, strings.Join(k.Enum(), ", "))
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
		if problem := checkOn(a.On); problem != "" {
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
