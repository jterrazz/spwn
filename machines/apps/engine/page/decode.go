package page

import (
	"encoding/json"
	"errors"
)

type Document struct {
	Schema string `json:"schema"`
	Page
}

func Decode(b []byte) (Page, error) {
	var d Document
	if err := json.Unmarshal(b, &d); err != nil {
		return Page{}, err
	}
	if d.Schema != Schema {
		return Page{}, errors.New("not a " + Schema + " document: " + d.Schema)
	}
	return d.Page, nil
}
