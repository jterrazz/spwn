package wiki

import (
	"bytes"
	"encoding/json"
)

type At struct {
	Line int `json:"line"`
	// value: 1-based, in runes.
	Col int `json:"col"`
}

type Span struct {
	// value: the first and the last line a record spans, 1-based.
	From int `json:"from"`
	To   int `json:"to"`
}

type Page struct {
	ID       string       `json:"id"`
	Faces    PageFaces    `json:"faces"`
	Kind     string       `json:"kind,omitempty"`
	Front    *Frontmatter `json:"frontmatter,omitempty"`
	Title    *Span        `json:"title,omitempty"`
	Sections []Section    `json:"sections"`
	Findings []Finding    `json:"findings"`
}

type PageFaces struct {
	Word      string `json:"word"`
	Line      string `json:"line"`
	Paragraph string `json:"paragraph"`
	Icon      string `json:"icon"`
}

type Frontmatter struct {
	Span  Span        `json:"span"`
	Lines []FrontLine `json:"lines"`
}

type FrontLine struct {
	Key string `json:"key"`
	// value: as written, its quotes dropped.
	Value string `json:"value"`
	Line  int    `json:"line"`
}

type Section struct {
	Heading *Heading `json:"heading"`
	Items   []Item   `json:"items"`
}

type Heading struct {
	Level  int    `json:"level"`
	Text   string `json:"text"`
	Anchor string `json:"anchor"`
	At     At     `json:"at"`
}

type Item interface {
	// invariant: an Item is a *Prose, a *Block or a *Group, and a Group holds no Group.
	item()
}

type Prose struct {
	T        string `json:"t"`
	Markdown string `json:"markdown"`
	Span     Span   `json:"span"`
}

type Group struct {
	T      string `json:"t"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Intent Intent `json:"intent"`
	Fold   bool   `json:"fold"`
	Props  Props  `json:"props"`
	Items  []Item `json:"items"`
	At     At     `json:"at"`
	Tag    Span   `json:"tag"`
	Span   Span   `json:"span"`
}

type Block struct {
	T       string     `json:"t"`
	ID      string     `json:"id"`
	Kind    string     `json:"kind"`
	Address string     `json:"address"`
	Props   Props      `json:"props"`
	Faces   BlockFaces `json:"faces"`
	View    View       `json:"view,omitempty"`
	At      At         `json:"at"`
	Span    Span       `json:"span"`
}

func (*Prose) item() {}
func (*Group) item() {}
func (*Block) item() {}

type BlockFaces struct {
	Word string `json:"word"`
	Line string `json:"line"`
	Icon string `json:"icon"`
}

type Intent string

const (
	IntentTell   Intent = "tell"
	IntentDefine Intent = "define"
	IntentDecide Intent = "decide"
	IntentLocate Intent = "locate"
	IntentGuard  Intent = "guard"
	IntentProve  Intent = "prove"
)

type View interface {
	// invariant: a View is a *TableView, a *SubPageView or a *FigureView.
	view()
}

type TableView struct {
	Of      string   `json:"of"`
	In      string   `json:"in"`
	Where   *Query   `json:"where"`
	Columns []string `json:"columns"`
	Sort    string   `json:"sort,omitempty"`
}

type SubPageView struct {
	Child string `json:"child"`
	Face  string `json:"face"`
}

type FigureView struct {
	Form    string       `json:"form"`
	Shows   string       `json:"shows"`
	Parts   []FigurePart `json:"parts"`
	Caption string       `json:"caption"`
}

type FigurePart struct {
	Name string
	To   string `json:",omitempty"`
	N    int    `json:",omitempty"`
	At   string `json:",omitempty"`
}

func (*TableView) view()   {}
func (*SubPageView) view() {}
func (*FigureView) view()  {}

type Op string

const (
	OpHas Op = "has"
	OpIs  Op = "is"
)

type Query struct {
	Field   string `json:"field"`
	Op      Op     `json:"op"`
	Operand Value  `json:"operand"`
}

type ValueKind string

const (
	ValueString ValueKind = "string"
	ValueNumber ValueKind = "number"
	ValueBool   ValueKind = "bool"
	ValueNull   ValueKind = "null"
	ValueList   ValueKind = "list"
	ValueObject ValueKind = "object"
	ValueRef    ValueKind = "ref"
	ValueQuery  ValueKind = "query"
	ValueState  ValueKind = "state"
)

type Value struct {
	Kind ValueKind
	// invariant: a string's characters, a number as written, a ref's address
	// or a state's value; nothing in it was ever evaluated.
	Text    string
	Bool    bool
	List    []Value
	Members []Member
	Query   *Query
	At      At
}

type Member struct {
	Key   string
	Value Value
}

type Prop struct {
	Name  string
	Value Value
	At    At
}

type Props []Prop

func (p Props) Get(name string) (Value, bool) {
	for _, prop := range p {
		if prop.Name == name {
			return prop.Value, true
		}
	}
	return Value{}, false
}

func (p Props) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, prop := range p {
		if i > 0 {
			b.WriteByte(',')
		}
		if err := writeKey(&b, prop.Name, prop.Value); err != nil {
			return nil, err
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func writeKey(b *bytes.Buffer, key string, v Value) error {
	k, err := json.Marshal(key)
	if err != nil {
		return err
	}
	b.Write(k)
	b.WriteByte(':')
	body, err := v.MarshalJSON()
	if err != nil {
		return err
	}
	b.Write(body)
	return nil
}

func (v Value) MarshalJSON() ([]byte, error) {
	switch v.Kind {
	case ValueString:
		return json.Marshal(v.Text)
	case ValueNumber:
		return []byte(v.Text), nil
	case ValueBool:
		return json.Marshal(v.Bool)
	case ValueList:
		list := v.List
		if list == nil {
			list = []Value{}
		}
		return json.Marshal(list)
	case ValueObject:
		var b bytes.Buffer
		b.WriteByte('{')
		for i, m := range v.Members {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeKey(&b, m.Key, m.Value); err != nil {
				return nil, err
			}
		}
		b.WriteByte('}')
		return b.Bytes(), nil
	case ValueRef:
		return json.Marshal(map[string]string{"ref": v.Text})
	case ValueState:
		return json.Marshal(map[string]string{"state": v.Text})
	case ValueQuery:
		return json.Marshal(map[string]*Query{"query": v.Query})
	default:
		return []byte("null"), nil
	}
}

func (q Query) MarshalJSON() ([]byte, error) {
	operand := map[string]string{"literal": q.Operand.Text}
	switch q.Operand.Kind {
	case ValueRef:
		operand = map[string]string{"ref": q.Operand.Text}
	case ValueState:
		operand = map[string]string{"state": q.Operand.Text}
	default:
	}
	return json.Marshal(struct {
		Field   string            `json:"field"`
		Op      Op                `json:"op"`
		Operand map[string]string `json:"operand"`
	}{q.Field, q.Op, operand})
}

type Finding struct {
	Rule  string `json:"rule"`
	Hard  bool   `json:"hard"`
	Level Level  `json:"level,omitempty"`
	Page  string `json:"page"`
	Block string `json:"block,omitempty"`
	Line  int    `json:"line"`
	Col   int    `json:"col"`
	Says  string `json:"says"`
}
