package catalog

import (
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v3"
)

// value: the schema of a wiki corpus's `wiki.yaml`, and the family its
// versions share.

const (
	WikiSchema = "spwn.wiki/v1"
	WikiFamily = "spwn.wiki"
)

var wikis = format{key: "wiki", schema: WikiSchema, family: WikiFamily}

type WikiFile struct {
	Wiki    string        `yaml:"wiki"`
	Floors  []WikiFloor   `yaml:"floors"`
	FloorOf []WikiFloorOf `yaml:"floor_of"`
	Seats   []WikiSeat    `yaml:"seats"`
	Mounts  []WikiMount   `yaml:"mounts"`
	Repos   []WikiRepo    `yaml:"repos"`
	Budget  string        `yaml:"budget"`
	Kinds   []WikiKind    `yaml:"kinds"`
	Icons   []WikiIcon    `yaml:"icons"`
	// value: the line of each top-level key, by its name.
	Lines map[string]int `yaml:"-"`
}

type WikiFloor struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Question string `yaml:"question"`
	Line     int    `yaml:"-"`
}

type WikiFloorOf struct {
	Path  string
	Floor int
	Line  int
}

type WikiSeat struct {
	ID    string   `yaml:"id"`
	Name  string   `yaml:"name"`
	Path  []string `yaml:"path"`
	Reads []string `yaml:"reads"`
	Home  string   `yaml:"home"`
	Plan  bool     `yaml:"plan"`
	Line  int      `yaml:"-"`
}

type WikiMount struct {
	At   string `yaml:"at"`
	Repo string `yaml:"repo"`
	Path string `yaml:"path"`
	Line int    `yaml:"-"`
}

type WikiRepo struct {
	Name string
	Path string
	Line int
}

type WikiKind struct {
	ID     string            `yaml:"-"`
	Line   int               `yaml:"-"`
	Title  map[string]string `yaml:"title"`
	Icon   string            `yaml:"icon"`
	Weighs int               `yaml:"weighs"`
	State  string            `yaml:"state"`
	Fields []WikiField       `yaml:"fields"`
	Faces  WikiFaces         `yaml:"faces"`
}

type WikiField struct {
	Name     string   `yaml:"-"`
	Line     int      `yaml:"-"`
	Type     string   `yaml:"type"`
	Max      int      `yaml:"max"`
	Required bool     `yaml:"required"`
	Values   []string `yaml:"values"`
	To       string   `yaml:"to"`
}

type WikiFaces struct {
	Word string `yaml:"word"`
	Line string `yaml:"line"`
	Icon string `yaml:"icon"`
}

type WikiIcon struct {
	Name string
	Rows []string
	Line int
}

func DecodeWiki(file string, body []byte) (WikiFile, error) {
	// invariant: wiki.yaml is read as strictly as a Machinefile: its schema
	// first, one document, every key known at every depth; a mapping keeps
	// the order it is written in.
	var w WikiFile
	doc, err := strict(file, body, wikis, &w)
	if err != nil {
		return WikiFile{}, err
	}
	w.Lines = map[string]int{}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		w.Lines[doc.Content[i].Value] = doc.Content[i].Line
	}
	for i := range w.Floors {
		w.Floors[i].Line = itemLine(doc, "floors", i)
	}
	for i := range w.Seats {
		w.Seats[i].Line = itemLine(doc, "seats", i)
	}
	for i := range w.Mounts {
		w.Mounts[i].Line = itemLine(doc, "mounts", i)
	}
	return w, nil
}

func pairsOf(n *yaml.Node, what string) ([][2]*yaml.Node, error) {
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: %s is a mapping of names", n.Line, what)
	}
	out := make([][2]*yaml.Node, 0, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		out = append(out, [2]*yaml.Node{n.Content[i], n.Content[i+1]})
	}
	return out, nil
}

type (
	wikiFloorOfs []WikiFloorOf
	wikiRepos    []WikiRepo
	wikiKinds    []WikiKind
	wikiFields   []WikiField
	wikiIcons    []WikiIcon
)

func (w *WikiFile) UnmarshalYAML(n *yaml.Node) error {
	type plain struct {
		Wiki    string       `yaml:"wiki"`
		Floors  []WikiFloor  `yaml:"floors"`
		FloorOf wikiFloorOfs `yaml:"floor_of"`
		Seats   []WikiSeat   `yaml:"seats"`
		Mounts  []WikiMount  `yaml:"mounts"`
		Repos   wikiRepos    `yaml:"repos"`
		Budget  string       `yaml:"budget"`
		Kinds   wikiKinds    `yaml:"kinds"`
		Icons   wikiIcons    `yaml:"icons"`
	}
	if n.Kind == yaml.MappingNode {
		if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
			return err
		}
	}
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	*w = WikiFile{Wiki: p.Wiki, Floors: p.Floors, FloorOf: p.FloorOf, Seats: p.Seats, Mounts: p.Mounts, Repos: p.Repos, Budget: p.Budget, Kinds: p.Kinds, Icons: p.Icons}
	return nil
}

func (f *WikiFloor) UnmarshalYAML(n *yaml.Node) error {
	type plain WikiFloor
	if n.Kind == yaml.MappingNode {
		if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
			return err
		}
	}
	return n.Decode((*plain)(f))
}

func (s *WikiSeat) UnmarshalYAML(n *yaml.Node) error {
	type plain WikiSeat
	if n.Kind == yaml.MappingNode {
		if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
			return err
		}
	}
	return n.Decode((*plain)(s))
}

func (m *WikiMount) UnmarshalYAML(n *yaml.Node) error {
	type plain WikiMount
	if n.Kind == yaml.MappingNode {
		if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
			return err
		}
	}
	return n.Decode((*plain)(m))
}

func (f *wikiFloorOfs) UnmarshalYAML(n *yaml.Node) error {
	pairs, err := pairsOf(n, "floor_of")
	if err != nil {
		return err
	}
	out := wikiFloorOfs{}
	for _, p := range pairs {
		var floor int
		if err := p[1].Decode(&floor); err != nil {
			return fmt.Errorf("line %d: the floor of %s is a number", p[1].Line, p[0].Value)
		}
		out = append(out, WikiFloorOf{Path: p[0].Value, Floor: floor, Line: p[0].Line})
	}
	*f = out
	return nil
}

func (r *wikiRepos) UnmarshalYAML(n *yaml.Node) error {
	pairs, err := pairsOf(n, "repos")
	if err != nil {
		return err
	}
	out := wikiRepos{}
	for _, p := range pairs {
		if p[1].Kind != yaml.ScalarNode {
			return fmt.Errorf("line %d: the repository %s is a path", p[1].Line, p[0].Value)
		}
		out = append(out, WikiRepo{Name: p[0].Value, Path: p[1].Value, Line: p[0].Line})
	}
	*r = out
	return nil
}

func (k *wikiKinds) UnmarshalYAML(n *yaml.Node) error {
	pairs, err := pairsOf(n, "kinds")
	if err != nil {
		return err
	}
	out := wikiKinds{}
	for _, p := range pairs {
		var kind WikiKind
		if err := p[1].Decode(&kind); err != nil {
			return err
		}
		kind.ID, kind.Line = p[0].Value, p[0].Line
		out = append(out, kind)
	}
	*k = out
	return nil
}

func (k *WikiKind) UnmarshalYAML(n *yaml.Node) error {
	type plain struct {
		Title  map[string]string `yaml:"title"`
		Icon   string            `yaml:"icon"`
		Weighs int               `yaml:"weighs"`
		State  string            `yaml:"state"`
		Fields wikiFields        `yaml:"fields"`
		Faces  WikiFaces         `yaml:"faces"`
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a kind is a mapping of its keys", n.Line)
	}
	if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
		return err
	}
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	*k = WikiKind{Title: p.Title, Icon: p.Icon, Weighs: p.Weighs, State: p.State, Fields: p.Fields, Faces: p.Faces}
	return nil
}

func (f *wikiFields) UnmarshalYAML(n *yaml.Node) error {
	pairs, err := pairsOf(n, "fields")
	if err != nil {
		return err
	}
	out := wikiFields{}
	for _, p := range pairs {
		var field WikiField
		if err := p[1].Decode(&field); err != nil {
			return err
		}
		field.Name, field.Line = p[0].Value, p[0].Line
		out = append(out, field)
	}
	*f = out
	return nil
}

func (f *WikiField) UnmarshalYAML(n *yaml.Node) error {
	type plain WikiField
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a field is a mapping of its keys (`{ type: sentence }`)", n.Line)
	}
	if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
		return err
	}
	return n.Decode((*plain)(f))
}

func (f *WikiFaces) UnmarshalYAML(n *yaml.Node) error {
	type plain WikiFaces
	if n.Kind == yaml.MappingNode {
		if err := knownKeys(n, reflect.TypeFor[plain]()); err != nil {
			return err
		}
	}
	return n.Decode((*plain)(f))
}

func (i *wikiIcons) UnmarshalYAML(n *yaml.Node) error {
	pairs, err := pairsOf(n, "icons")
	if err != nil {
		return err
	}
	out := wikiIcons{}
	for _, p := range pairs {
		var rows []string
		if err := p[1].Decode(&rows); err != nil {
			return fmt.Errorf("line %d: the icon %s is a list of rows", p[1].Line, p[0].Value)
		}
		out = append(out, WikiIcon{Name: p[0].Value, Rows: rows, Line: p[0].Line})
	}
	*i = out
	return nil
}

// value: the line of the i-th item of a top-level list, 0 when it has none.
func itemLine(doc *yaml.Node, key string, i int) int {
	if _, v := entry(doc, key); v != nil && i < len(v.Content) {
		return v.Content[i].Line
	}
	return 0
}
