package wiki

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type Level string

const (
	LevelAmbient  Level = "ambient"
	LevelRemark   Level = "remark"
	LevelNeedsYou Level = "needs-you"
)

// invariant: the rules this engine judges, SPEC-v1 §17, each once: a hard
// rule has no level, a soft one its Signal's. A finding is made only through
// this table, so a code missing from it is never raised.
func rules() map[string]Level {
	return map[string]Level{
		"H1": "", "H2": "", "H3": "", "H4": "", "H5": "", "H6": "", "H8": "", "H9": "", "H10": "",
		"H12": "", "H14": "", "H15": "", "H16": "", "H17": "", "H19": "", "H20": "", "H22": "",
		"H23": "", "H24": "", "H25": "", "H26": "", "H28": "", "H29": "", "H30": "", "H31": "",
		"H32": "", "H33": "", "H35": "", "H36": "", "H38": "", "H39": "", "H40": "", "H41": "", "H44": "",
		"S2": LevelRemark, "S3": LevelRemark, "S4": LevelAmbient, "S6": LevelNeedsYou,
		"S8": LevelAmbient, "S11": LevelAmbient, "S12": LevelRemark, "S13": LevelAmbient,
		"S14": LevelAmbient, "S15": LevelRemark, "S16": LevelAmbient, "S17": LevelRemark,
		"S18": LevelAmbient, "S21": LevelAmbient, "S22": LevelRemark, "S26": LevelRemark,
		"S28": LevelRemark, "S29": LevelNeedsYou, "S30": LevelRemark, "S31": LevelAmbient, "S32": LevelRemark,
	}
}

// invariant: the table is built once, as a page of half a million findings
// looks it up for each; no caller writes to it.
var ruleLevels = sync.OnceValue(rules)

// value: the finding of a rule, hard when the rule has no level.
func finding(rule, page, block string, at At, says string) Finding {
	level, ok := ruleLevels()[rule]
	if !ok {
		panic("wiki: no rule " + rule)
	}
	f := Finding{Rule: rule, Hard: level == "", Level: level, Page: page, Block: block, Line: at.Line, Col: at.Col}
	f.Says, f.Cut = clip(says, MaxSaysBytes)
	return f
}

// value: s cut to its first n bytes, on a character's boundary when one is
// within a character's width, copied so it holds nothing of s; and whether
// it was cut.
func clip(s string, n int) (string, bool) {
	if len(s) <= n {
		return s, false
	}
	end := n
	for end > n-utf8.UTFMax && !utf8.RuneStart(s[end]) {
		end--
	}
	return strings.Clone(s[:end]), true
}

func Rules() []string {
	// value: the codes of the registry, hard first, each in its number's order.
	out := make([]string, 0, len(ruleLevels()))
	for code := range ruleLevels() {
		out = append(out, code)
	}
	slices.SortFunc(out, func(a, b string) int {
		if a[0] != b[0] {
			return cmp.Compare(a[0], b[0])
		}
		x, _ := strconv.Atoi(a[1:])
		y, _ := strconv.Atoi(b[1:])
		return cmp.Compare(x, y)
	})
	return out
}
