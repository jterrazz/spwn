package json

import (
	"bytes"
	"encoding/json"

	"github.com/jterrazz/studio-station/apps/engine/page"
)

func Render(p page.Page) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(page.Document{Schema: page.Schema, Page: p}); err != nil {
		panic(err)
	}
	return b.Bytes()
}
