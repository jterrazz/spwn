package catalog

import "strings"

type MacModel struct {
	// value: SPEC B5.12: the name a Mac model is shown by, keyed by the
	// identifier hw.model reports; Prefix matches a family of identifiers
	// (the Intel and first Apple silicon ones carry their family's name).
	ID, Prefix, Name string
}

var MacModels = []MacModel{
	{ID: "Mac14,2", Name: "MacBook Air"}, {ID: "Mac14,15", Name: "MacBook Air"},
	{ID: "Mac15,12", Name: "MacBook Air"}, {ID: "Mac15,13", Name: "MacBook Air"},
	{ID: "Mac16,12", Name: "MacBook Air"}, {ID: "Mac16,13", Name: "MacBook Air"},
	{ID: "Mac14,5", Name: "MacBook Pro"}, {ID: "Mac14,6", Name: "MacBook Pro"}, {ID: "Mac14,7", Name: "MacBook Pro"},
	{ID: "Mac14,9", Name: "MacBook Pro"}, {ID: "Mac14,10", Name: "MacBook Pro"},
	{ID: "Mac15,3", Name: "MacBook Pro"}, {ID: "Mac15,6", Name: "MacBook Pro"}, {ID: "Mac15,7", Name: "MacBook Pro"},
	{ID: "Mac15,8", Name: "MacBook Pro"}, {ID: "Mac15,9", Name: "MacBook Pro"}, {ID: "Mac15,10", Name: "MacBook Pro"},
	{ID: "Mac15,11", Name: "MacBook Pro"},
	{ID: "Mac16,1", Name: "MacBook Pro"}, {ID: "Mac16,5", Name: "MacBook Pro"}, {ID: "Mac16,6", Name: "MacBook Pro"},
	{ID: "Mac16,7", Name: "MacBook Pro"}, {ID: "Mac16,8", Name: "MacBook Pro"},
	{ID: "Mac14,3", Name: "Mac mini"}, {ID: "Mac14,12", Name: "Mac mini"},
	{ID: "Mac16,10", Name: "Mac mini"}, {ID: "Mac16,11", Name: "Mac mini"},
	{ID: "Mac15,4", Name: "iMac"}, {ID: "Mac15,5", Name: "iMac"}, {ID: "Mac16,2", Name: "iMac"}, {ID: "Mac16,3", Name: "iMac"},
	{ID: "Mac13,1", Name: "Mac Studio"}, {ID: "Mac13,2", Name: "Mac Studio"}, {ID: "Mac14,13", Name: "Mac Studio"},
	{ID: "Mac14,14", Name: "Mac Studio"}, {ID: "Mac15,14", Name: "Mac Studio"}, {ID: "Mac16,9", Name: "Mac Studio"},
	{ID: "Mac14,8", Name: "Mac Pro"},
	{Prefix: "MacBookAir", Name: "MacBook Air"},
	{Prefix: "MacBookPro", Name: "MacBook Pro"},
	{Prefix: "MacBook", Name: "MacBook"},
	{Prefix: "Macmini", Name: "Mac mini"},
	{Prefix: "iMac", Name: "iMac"},
	{Prefix: "MacPro", Name: "Mac Pro"},
}

func MacModelName(id string) string {
	// value: the name a Mac is shown by, from its hw.model: "Mac" for an
	// identifier the table does not hold. It is shown, never read by a rule.
	for _, m := range MacModels {
		if m.ID != "" && m.ID == id {
			return m.Name
		}
	}
	for _, m := range MacModels {
		if m.Prefix != "" && strings.HasPrefix(id, m.Prefix) {
			return m.Name
		}
	}
	return "Mac"
}
