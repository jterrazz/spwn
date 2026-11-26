package architecture_test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type inputField struct{ list, text bool }

type callInput struct {
	fields map[string]inputField
	// value: whether its confirmation is typed for some kind of machine,
	// and for every kind.
	typedSome, typedAll bool
}

func callInputs(root string) (map[string]callInput, error) {
	var list struct {
		Calls []struct{ Name string } `json:"calls"`
	}
	if err := readJSON(filepath.Join(root, "apps", "engine", "remote", "api", "calls.json"), &list); err != nil {
		return nil, err
	}
	type form struct {
		Form string `json:"form"`
	}
	var confirms map[string]struct {
		Form  string          `json:"form"`
		Kinds map[string]form `json:"kinds"`
	}
	if err := readJSON(filepath.Join(root, "packages", "api", "confirms.json"), &confirms); err != nil {
		return nil, err
	}
	out := map[string]callInput{}
	for _, c := range list.Calls {
		in, err := inputOf(root, c.Name)
		if err != nil {
			return nil, err
		}
		if k, ok := confirms[c.Name]; ok {
			forms := []string{k.Form}
			for _, f := range k.Kinds {
				forms = append(forms, f.Form)
			}
			in.typedSome = slices.Contains(forms, "type-name")
			in.typedAll = !slices.ContainsFunc(forms, func(f string) bool { return f != "type-name" })
		}
		out[c.Name] = in
	}
	return out, nil
}

func inputOf(root, name string) (callInput, error) {
	type property struct {
		Type  any `json:"type"`
		Items *struct {
			Type any `json:"type"`
		} `json:"items"`
	}
	var schema struct {
		Defs map[string]struct {
			Properties map[string]property `json:"properties"`
		} `json:"$defs"`
		Properties struct {
			In struct {
				Ref string `json:"$ref"`
			} `json:"in"`
		} `json:"properties"`
	}
	if err := readJSON(filepath.Join(root, "packages", "api", "schemas", "calls", name+".json"), &schema); err != nil {
		return callInput{}, err
	}
	in := callInput{fields: map[string]inputField{}}
	for key, p := range schema.Defs[strings.TrimPrefix(schema.Properties.In.Ref, "#/$defs/")].Properties {
		f := inputField{text: typed(p.Type, "string")}
		if p.Items != nil && typed(p.Type, "array") {
			f.list = typed(p.Items.Type, "string")
		}
		in.fields[key] = f
	}
	return in, nil
}

func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func typed(t any, want string) bool {
	switch v := t.(type) {
	case string:
		return v == want
	case []any:
		return slices.Contains(v, any(want))
	}
	return false
}

// invariant: rule fix-args-are-inputs: a finding's fix names its call's
// input as the call reads it. A Fix whose Call is a literal names a call of
// calls.json, and its Args are a literal map whose every key is a field of
// that call's input in its schema, a text or a list of texts (a list holds
// the one value given); a Fix that sets Confirm names a call whose question
// is typed for some machine, and one whose question is typed for every
// machine sets it. A Fix copied whole (Call and Args both not literal)
// carries what its source was judged on.
func ruleFixArgsAreInputs(w *world) []finding {
	ins, err := callInputs(w.root)
	if err != nil {
		return []finding{{"fix-args-are-inputs", "packages/api", err.Error()}}
	}
	var out []finding
	for _, f := range w.files {
		if f.test {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isFix(f.info.TypeOf(lit)) {
				return true
			}
			out = append(out, judgeFix(f, lit, ins)...)
			return true
		})
	}
	return out
}

func isFix(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Name() == "Fix" && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == module+"/core/finding"
}

func judgeFix(f *file, lit *ast.CompositeLit, ins map[string]callInput) []finding {
	fields := map[string]ast.Expr{}
	for _, e := range lit.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok {
				fields[id.Name] = kv.Value
			}
		}
	}
	bad := func(what string, args ...any) []finding {
		return []finding{{"fix-args-are-inputs", f.at(lit.Pos()), fmt.Sprintf(what, args...)}}
	}
	name, literal := stringLit(fields["Call"])
	args, isMap := fields["Args"].(*ast.CompositeLit)
	switch {
	case !literal && isMap:
		return bad("a fix's literal Args name a call by its literal name")
	case !literal:
		return nil
	}
	in, known := ins[name]
	if !known {
		return bad("%q is no call of calls.json", name)
	}
	var out []finding
	if a, set := fields["Args"]; set && !isMap {
		out = append(out, finding{"fix-args-are-inputs", f.at(a.Pos()), fmt.Sprintf("the Args of a %s fix are a literal map, so each key is judged", name)})
	}
	if isMap {
		for _, e := range args.Elts {
			kv, ok := e.(*ast.KeyValueExpr)
			key, isLit := stringLit(kv.Key)
			if !ok || !isLit {
				out = append(out, finding{"fix-args-are-inputs", f.at(e.Pos()), fmt.Sprintf("a %s fix's Args key is a literal", name)})
				continue
			}
			field, has := in.fields[key]
			switch {
			case !has:
				out = append(out, finding{"fix-args-are-inputs", f.at(kv.Pos()), fmt.Sprintf("%q is no field of %s's input: it takes %s", key, name, strings.Join(sortedFields(in), ", "))})
			case !field.text && !field.list:
				out = append(out, finding{"fix-args-are-inputs", f.at(kv.Pos()), fmt.Sprintf("%s's %s is neither a text nor a list of texts", name, key)})
			}
		}
	}
	_, confirms := fields["Confirm"]
	switch {
	case confirms && !in.typedSome:
		out = append(out, finding{"fix-args-are-inputs", f.at(lit.Pos()), fmt.Sprintf("a %s fix sets Confirm, but its question is never typed", name)})
	case !confirms && in.typedAll:
		out = append(out, finding{"fix-args-are-inputs", f.at(lit.Pos()), fmt.Sprintf("a %s fix leaves Confirm empty, but its question is always typed", name)})
	}
	return out
}

func stringLit(e ast.Expr) (string, bool) {
	b, ok := e.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(b.Value)
	return v, err == nil
}

func sortedFields(in callInput) []string {
	out := make([]string, 0, len(in.fields))
	for k := range in.fields {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func TestFixArgsAreInputs(t *testing.T)        { judge(t, "FixArgsAreInputs") }
func TestFixArgsAreInputsConfirm(t *testing.T) { judge(t, "FixArgsAreInputsConfirm") }

// invariant: rule fix-is-a-gesture: a fix is exactly one gesture the window
// can draw: a call, a pane, or a command the owner runs in a terminal
// (InTerminal); a Path alone is a sentence. A Fix literal that sets a
// Command with no Call says InTerminal, one that says InTerminal has a
// Command, and one that sets a Pane sets no Call, Command or InTerminal.
func ruleFixIsAGesture(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isFix(f.info.TypeOf(lit)) {
				return true
			}
			out = append(out, judgeGesture(f, lit)...)
			return true
		})
	}
	return out
}

func judgeGesture(f *file, lit *ast.CompositeLit) []finding {
	set := map[string]bool{}
	for _, e := range lit.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok {
				set[id.Name] = true
			}
		}
	}
	bad := func(what string) []finding {
		return []finding{{"fix-is-a-gesture", f.at(lit.Pos()), what}}
	}
	switch {
	case set["Command"] && !set["Call"] && !set["InTerminal"]:
		return bad("a fix that is a command and no call says InTerminal: the owner runs it in a terminal")
	case set["InTerminal"] && !set["Command"]:
		return bad("a fix that says InTerminal carries the Command it fills in")
	case set["Pane"] && (set["Call"] || set["Command"] || set["InTerminal"]):
		return bad("a fix that opens a pane is that one gesture alone")
	}
	return nil
}

func TestFixIsAGesture(t *testing.T) { judge(t, "FixIsAGesture") }
