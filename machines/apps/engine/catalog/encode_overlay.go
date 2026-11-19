package catalog

import (
	"bytes"
	"errors"
	"strconv"

	"go.yaml.in/yaml/v3"
)

var errNoOverlay = errors.New("an overlay names the file it changes in overrides, and carries apps, settings, accounts and ports alone")

func EncodeOverlay(mf Machinefile) ([]byte, error) {
	// invariant: station's one encoding of an overlay: the schema's keys in
	// its order (machinefile, overrides, apps, settings, accounts, ports),
	// each chapter on one line in flow style, its keys in the schema's
	// order too, no comment; DecodeMachinefile reads back the same value.
	if mf.Overrides == "" || mf.Name != "" || mf.Machine != nil || mf.VM != nil || len(mf.Extends) > 0 || len(mf.Prefer) > 0 || mf.Run != nil || mf.Description != "" {
		return nil, errNoOverlay
	}
	doc := mapping(false)
	put(doc, "machinefile", str(MachinefileSchema))
	put(doc, "overrides", str(mf.Overrides))
	if a := mf.Apps; a != nil {
		apps := mapping(true)
		if a.Add != nil {
			add := sequence()
			for _, e := range a.Add {
				add.Content = append(add.Content, appNode(e))
			}
			put(apps, "add", add)
		}
		if a.Remove != nil {
			put(apps, "remove", strs(a.Remove))
		}
		put(doc, "apps", apps)
	}
	if s := mf.Settings; s != nil {
		settings := mapping(true)
		if s.Set != nil {
			set := mapping(true)
			for _, name := range sortedKeys(s.Set) {
				put(set, name, settingNode(s.Set[name]))
			}
			put(settings, "set", set)
		}
		if s.Remove != nil {
			put(settings, "remove", strs(s.Remove))
		}
		put(doc, "settings", settings)
	}
	if a := mf.Accounts; a != nil {
		accounts := mapping(true)
		if a.SignIns != nil {
			put(accounts, "signins", strs(a.SignIns))
		}
		if a.Grants != nil {
			put(accounts, "grants", strs(a.Grants))
		}
		put(doc, "accounts", accounts)
	}
	if p := mf.Ports; p != nil {
		put(doc, "ports", portsNode(*p))
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(4)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func portsNode(p PortRights) *yaml.Node {
	ports := mapping(true)
	if p.Open != nil {
		open := sequence()
		for _, s := range p.Open {
			if s.From != s.To {
				open.Content = append(open.Content, str(s.String()))
			} else {
				open.Content = append(open.Content, integer(s.From))
			}
		}
		put(ports, "open", open)
	}
	if p.To != "" {
		put(ports, "to", str(string(p.To)))
	}
	if p.Except != nil {
		except := sequence()
		for _, port := range p.Except {
			except.Content = append(except.Content, integer(port))
		}
		put(ports, "except", except)
	}
	if p.Audiences != nil {
		audiences := mapping(true)
		for _, port := range p.Audienced() {
			audiences.Content = append(audiences.Content, integer(port), str(string(p.Audiences[port])))
		}
		put(ports, "audiences", audiences)
	}
	return ports
}

func appNode(e AppEntry) *yaml.Node {
	if e.Version.IsZero() && !e.Optional && e.Manager == "" && e.Options == nil {
		return str(e.Name)
	}
	app := mapping(true)
	put(app, "name", str(e.Name))
	if !e.Version.IsZero() {
		put(app, "version", str(e.Version.String()))
	}
	if e.Optional {
		put(app, "optional", boolean(true))
	}
	if e.Manager != "" {
		put(app, "manager", str(string(e.Manager)))
	}
	if e.Options != nil {
		options := mapping(true)
		for _, k := range sortedKeys(e.Options) {
			put(options, k, str(string(e.Options[k])))
		}
		put(app, "options", options)
	}
	return app
}

func settingNode(e SettingEntry) *yaml.Node {
	setting := mapping(true)
	if e.From != "" {
		put(setting, "from", str(e.From))
	}
	if e.Optional {
		put(setting, "optional", boolean(true))
	}
	for _, k := range sortedKeys(e.Values) {
		put(setting, k, str(string(e.Values[k])))
	}
	return setting
}

func mapping(flow bool) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if flow {
		n.Style = yaml.FlowStyle
	}
	return n
}

func put(m *yaml.Node, key string, value *yaml.Node) {
	m.Content = append(m.Content, str(key), value)
}

func sequence() *yaml.Node {
	return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
}

func strs[T ~string](list []T) *yaml.Node {
	s := sequence()
	for _, v := range list {
		s.Content = append(s.Content, str(string(v)))
	}
	return s
}

func str(v string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
	if yaml11Bool.MatchString(v) || yaml11Octal.MatchString(v) {
		n.Style = yaml.SingleQuotedStyle
	}
	return n
}

func integer(v int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(v)}
}

func boolean(v bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(v)}
}
