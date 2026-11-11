package catalog

import (
	"strconv"
	"strings"
)

type Method string

const (
	MethodBrew   Method = "brew"
	MethodCask   Method = "cask"
	MethodMas    Method = "mas"
	MethodNpm    Method = "npm"
	MethodUv     Method = "uv"
	MethodBun    Method = "bun"
	MethodMise   Method = "mise"
	MethodScript Method = "script"
	MethodShell  Method = "shell"
	MethodApp    Method = "app"
	MethodSkill  Method = "skill"
	// invariant: the command comes with another item's install: installing it
	// installs that item, and its presence is its own provides.
	MethodProvidedBy Method = "provided-by"
)

type Install interface {
	Method() Method
	Key() string
	isInstall()
}

type BrewInstall struct {
	Formula string
	Trusted bool
}

type CaskInstall struct {
	Token   string
	Trusted bool
}

type MasInstall struct {
	Name string
	ID   int
}

type NpmInstall struct{ Package string }

type UvInstall struct{ Package string }

type BunInstall struct{ Package, Name string }

type MiseInstall struct{ App, Version string }

type ScriptInstall struct {
	URL  string
	Args []string
	Env  []string
	// value: the shell line that updates what the installer put, for an installer
	// that refuses to run over its own install; empty, an update runs the installer.
	Upgrade string
	// value: the variables added when the run has a terminal, and when it has
	// none: an installer that asks there and keeps quiet elsewhere.
	OnTerminal, OffTerminal []string
}

type ShellInstall struct{ Line string }

type AppInstall struct{ Hint string }

type SkillInstall struct{ Repo, Skill string }

type ProvidedByInstall struct {
	Item string
	// value: the commands the item brings, which its install finds.
	Commands []string
}

func (BrewInstall) Method() Method       { return MethodBrew }
func (CaskInstall) Method() Method       { return MethodCask }
func (MasInstall) Method() Method        { return MethodMas }
func (NpmInstall) Method() Method        { return MethodNpm }
func (UvInstall) Method() Method         { return MethodUv }
func (BunInstall) Method() Method        { return MethodBun }
func (MiseInstall) Method() Method       { return MethodMise }
func (ScriptInstall) Method() Method     { return MethodScript }
func (ShellInstall) Method() Method      { return MethodShell }
func (AppInstall) Method() Method        { return MethodApp }
func (SkillInstall) Method() Method      { return MethodSkill }
func (ProvidedByInstall) Method() Method { return MethodProvidedBy }

func (i BrewInstall) Key() string { return lastSegment(i.Formula) }
func (i CaskInstall) Key() string { return lastSegment(i.Token) }
func (i MasInstall) Key() string  { return strconv.Itoa(i.ID) }
func (i NpmInstall) Key() string  { return i.Package }

func (i UvInstall) Key() string {
	name, _, _ := strings.Cut(i.Package, "[")
	return name
}

func (i BunInstall) Key() string {
	if i.Name != "" {
		return i.Name
	}
	return i.Package
}

func (i MiseInstall) Key() string       { return i.App }
func (i ScriptInstall) Key() string     { return i.URL }
func (i ShellInstall) Key() string      { return i.Line }
func (i AppInstall) Key() string        { return i.Hint }
func (i SkillInstall) Key() string      { return i.Skill }
func (i ProvidedByInstall) Key() string { return i.Item }

func (BrewInstall) isInstall()       {}
func (CaskInstall) isInstall()       {}
func (MasInstall) isInstall()        {}
func (NpmInstall) isInstall()        {}
func (UvInstall) isInstall()         {}
func (BunInstall) isInstall()        {}
func (MiseInstall) isInstall()       {}
func (ScriptInstall) isInstall()     {}
func (ShellInstall) isInstall()      {}
func (AppInstall) isInstall()        {}
func (SkillInstall) isInstall()      {}
func (ProvidedByInstall) isInstall() {}

func Pinned(in Install, version string) (Install, bool) {
	// value: the install at the version asked, where it carries one to its
	// manager: a runtime of mise's; any other carries none.
	if m, ok := in.(MiseInstall); ok {
		m.Version = version
		return m, true
	}
	return nil, false
}

func VersionOf(in Install) string {
	// value: the version an install carries to its manager, empty where
	// it carries none.
	if m, ok := in.(MiseInstall); ok {
		return m.Version
	}
	return ""
}

func lastSegment(name string) string {
	return name[strings.LastIndex(name, "/")+1:]
}

// invariant: a skill item is named after its skill, except where that name is
// already an app's: then it carries the `-skill` suffix, and its install keeps
// the upstream name the skills lock records.

// value: the agents a skill is installed for: Claude Code (~/.claude/skills) and
// the shared ~/.agents/skills that codex and the other universal agents read.
// Without it the skills CLI, finding no agent on a fresh home, writes to all 79.

var SkillAgents = []string{"claude-code", "codex"}
