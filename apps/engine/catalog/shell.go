package catalog

import (
	"fmt"
	"regexp"
	"strings"
)

// value: one identified block of ~/.zshrc a method writes: its id, which is
// stable for life (`station:<item>` or `station:<item>/<sub>`) and names the
// block's markers, and its plain zsh, with station's variables between double
// braces filled in when it is written.

type ShellBlock struct {
	ID  string `yaml:"id"`
	Zsh string `yaml:"zsh"`
}

// value: the blocks a setting's method writes, as its effect.

type ShellBlocks struct {
	Blocks []ShellBlock
}

func (ShellBlocks) Kind() EffectKind { return EffectShell }

// value: what a block's variables are filled from at apply: the machine, and
// the values the package's options hold.

type ShellVars struct {
	Home    string
	OS      OS
	Arch    Arch
	Options map[string]OptionValue
}

// value: the variables a block may name, besides `option.<name>`.
const (
	VarBrewPrefix = "brew.prefix"
	VarHome       = "home"
	VarOS         = "os"
	VarArch       = "arch"
	varOption     = "option."
)

func ShellVarNames() []string {
	return []string{VarBrewPrefix, VarHome, VarOS, VarArch, varOption + "<name>"}
}

func BrewPrefix(os OS, arch Arch) string {
	// value: Homebrew's prefix on a system: /opt/homebrew on an Apple silicon
	// Mac, /usr/local on an Intel one, Linuxbrew's elsewhere.
	switch {
	case os == OSMacOS && arch == ArchAMD64:
		return "/usr/local"
	case os == OSMacOS:
		return "/opt/homebrew"
	}
	return "/home/linuxbrew/.linuxbrew"
}

var shellVar = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

func (it Package) RenderShell(b ShellBlock, v ShellVars) (string, error) {
	// invariant: a variable is plain text substitution of a fixed set of names:
	// a name outside it, an option with no value, or a brace pair that is no
	// variable is an error, never an empty string, and nothing is evaluated.
	var out strings.Builder
	last := 0
	for _, m := range shellVar.FindAllStringSubmatchIndex(b.Zsh, -1) {
		out.WriteString(b.Zsh[last:m[0]])
		value, err := it.shellValue(b, strings.TrimSpace(b.Zsh[m[2]:m[3]]), v)
		if err != nil {
			return "", err
		}
		out.WriteString(value)
		last = m[1]
	}
	out.WriteString(b.Zsh[last:])
	if strings.Contains(shellVar.ReplaceAllString(b.Zsh, ""), "{{") {
		return "", fmt.Errorf("%s: a {{ is no variable", b.ID)
	}
	if text := out.String(); strings.HasSuffix(text, "\n") {
		return text, nil
	}
	return out.String() + "\n", nil
}

func (it Package) shellValue(b ShellBlock, name string, v ShellVars) (string, error) {
	switch {
	case name == VarBrewPrefix:
		return BrewPrefix(v.OS, v.Arch), nil
	case name == VarHome:
		return v.Home, nil
	case name == VarOS:
		return string(v.OS), nil
	case name == VarArch:
		return string(v.Arch), nil
	case strings.HasPrefix(name, varOption):
		opt := strings.TrimPrefix(name, varOption)
		if value, ok := v.Options[opt]; ok {
			return string(value), nil
		}
		for _, o := range it.Options {
			if o.Name == opt && o.Default != nil {
				return string(*o.Default), nil
			}
		}
		return "", fmt.Errorf("%s: option %s has no value", b.ID, opt)
	}
	return "", fmt.Errorf("%s: {{ %s }} is none of %s", b.ID, name, strings.Join(ShellVarNames(), ", "))
}

var shellID = regexp.MustCompile(`^station:([a-z0-9][a-z0-9.-]*)(/[a-z0-9][a-z0-9-]*)?$`)

// invariant: a block's id names its own item, so two items never write the
// same block; its zsh says something, never holds a marker line, and names
// only variables the machine or the item's non-secret options can fill (a
// setting's takes the machine's alone: options are an app's).
func (it Package) checkShell(blocks []ShellBlock) string {
	seen := map[string]bool{}
	for _, b := range blocks {
		m := shellID.FindStringSubmatch(b.ID)
		switch {
		case m == nil:
			return fmt.Sprintf("shell id %q is `station:<item>` or `station:<item>/<sub>`", b.ID)
		case m[1] != it.Name:
			return fmt.Sprintf("shell id %q names another item than %s", b.ID, it.Name)
		case seen[b.ID]:
			return fmt.Sprintf("shell id %q is written twice", b.ID)
		case strings.TrimSpace(b.Zsh) == "":
			return fmt.Sprintf("shell %s has no zsh", b.ID)
		}
		seen[b.ID] = true
		for _, line := range strings.Split(b.Zsh, "\n") {
			if t := strings.TrimSpace(line); strings.HasPrefix(t, "# >>> station") || strings.HasPrefix(t, "# <<< station") {
				return fmt.Sprintf("shell %s holds a marker line", b.ID)
			}
		}
		if problem := it.checkShellVars(b); problem != "" {
			return problem
		}
	}
	return ""
}

func (it Package) checkShellVars(b ShellBlock) string {
	for _, m := range shellVar.FindAllString(b.Zsh, -1) {
		name := strings.TrimSpace(m[2 : len(m)-2])
		switch {
		case name == VarBrewPrefix || name == VarHome || name == VarOS || name == VarArch:
		case strings.HasPrefix(name, varOption):
			opt := strings.TrimPrefix(name, varOption)
			switch {
			case it.Kind == KindSetting:
				return fmt.Sprintf("shell %s: a setting's block takes the machine's variables, not %s", b.ID, m)
			case !hasOption(it.Options, opt):
				return fmt.Sprintf("shell %s: %s names no option of the item", b.ID, m)
			}
			for _, o := range it.Options {
				if o.Name == opt && o.Secret {
					return fmt.Sprintf("shell %s: option %s is secret, never written to ~/.zshrc", b.ID, opt)
				}
			}
		default:
			return fmt.Sprintf("shell %s: {{ %s }} is none of %s", b.ID, name, strings.Join(ShellVarNames(), ", "))
		}
	}
	if strings.Contains(shellVar.ReplaceAllString(b.Zsh, ""), "{{") {
		return fmt.Sprintf("shell %s: a {{ is no variable", b.ID)
	}
	return ""
}
