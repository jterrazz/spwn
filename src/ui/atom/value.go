package atom

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ui/token"
)

const versionMax = 18

func Plain(v page.Value) string {
	switch v.Kind {
	case page.KindPortMap:
		return strconv.FormatInt(v.N, 10)
	case page.KindVersion:
		return v.Text
	case page.KindList:
		return join(v.Items, Plain)
	case page.KindLevels:
		parts := make([]string, 0, len(v.Items))
		for _, l := range v.Items {
			parts = append(parts, l.Label+" "+Plain(l))
		}
		return strings.Join(parts, token.Sep)
	case page.KindPending:
		return ""
	case page.KindSpark:
		return Plain(sparkNow(v))
	case page.KindUntil:
		if v.At != "" {
			return v.At
		}
	default:
	}
	return show(v)
}

// value: what a row says after its cells: its detail, then the time it
// has left, as one value.
func Detail(r page.Row) page.Value {
	if r.Left == nil {
		return r.Detail
	}
	if r.Detail.Kind == "" {
		return *r.Left
	}
	parts := []page.Value{r.Detail}
	if r.Detail.Kind == page.KindList {
		parts = append([]page.Value{}, r.Detail.Items...)
	}
	return page.List(append(parts, *r.Left)...)
}

func Show(v page.Value) string {
	switch v.Kind {
	case page.KindVersion:
		if v.Label != "" {
			return End(v.Text, versionMax) + " " + token.Arrow + " " + End(v.Label, versionMax)
		}
		return End(v.Text, versionMax)
	case page.KindPortMap:
		return strconv.FormatInt(v.N, 10) + " " + token.Arrow + " " + strconv.FormatInt(v.Of, 10)
	case page.KindList:
		return join(v.Items, Show)
	case page.KindCommand:
		return token.Arrow + " " + v.Text
	case page.KindPending:
		return ""
	case page.KindSpark:
		samples := make([]int, 0, len(v.Items))
		for _, x := range v.Items[min(1, len(v.Items)):] {
			samples = append(samples, int(x.N))
		}
		return Spark(samples) + " " + Show(sparkNow(v))
	default:
	}
	return show(v)
}

func sparkNow(v page.Value) page.Value {
	if len(v.Items) == 0 {
		return page.None()
	}
	return v.Items[0]
}

func show(v page.Value) string {
	// invariant: a value labeled with a mark leads with its glyph (an
	// Overview's `● 39 ok`), but a bare count, which reads as a title's
	// counts do, `47 ●`.
	if g := Glyph(page.Mark(v.Label)); g != "" && v.Kind != page.KindLevels && (v.Kind != page.KindCount || v.Unit != "") {
		v.Label = ""
		if rest := show(v); rest != "" {
			return g + " " + rest
		}
		return g
	}
	switch v.Kind {
	case page.KindNone, "", page.KindPending, page.KindListing:
		return ""
	case page.KindSpark:
		return Show(v)
	case page.KindCount:
		n := strconv.FormatInt(v.N, 10)
		if g := Glyph(page.Mark(v.Label)); g != "" && v.Unit == "" {
			return n + " " + g
		}
		if v.Unit == "" {
			return n
		}
		return n + " " + v.Unit
	case page.KindSize:
		s := Bytes(v.N)
		switch {
		case strings.HasPrefix(v.Unit, "/"):
			s += v.Unit
		case v.Unit != "":
			s += " " + v.Unit
		}
		if v.Of > 0 {
			s += " of " + Bytes(v.Of)
		}
		return s
	case page.KindRatio:
		used, total := unitOf(v.Of)
		return used.of(v.N) + "/" + total
	case page.KindPercent:
		if v.N < 100 && v.N%10 != 0 {
			return strconv.FormatInt(v.N/10, 10) + "." + strconv.FormatInt(v.N%10, 10) + "%"
		}
		return strconv.FormatInt(v.N/10, 10) + "%"
	case page.KindDuration:
		return Duration(v.N)
	case page.KindAge:
		if v.N > 30*86400 && len(v.At) >= 10 {
			return v.At[:10]
		}
		ago := Duration(v.N)[:durationHead(v.N)] + " ago"
		if v.Label != "" {
			return v.Label + " " + ago
		}
		return ago
	case page.KindUntil:
		switch {
		case v.N <= 0:
			return "expired"
		case v.N < 60:
			return "<1m left"
		}
		return Duration(v.N-v.N%60) + " left"
	case page.KindPort:
		return strconv.FormatInt(v.N, 10)
	case page.KindRange:
		return strconv.FormatInt(v.N, 10) + "-" + strconv.FormatInt(v.Of, 10)
	case page.KindOn:
		return "on"
	case page.KindOff:
		return "off"
	case page.KindUnknown:
		return token.Unknown
	case page.KindLevels:
		return Plain(v)
	case page.KindList:
		return join(v.Items, show)
	case page.KindText, page.KindVersion, page.KindURL, page.KindIdentity, page.KindPath, page.KindCommand, page.KindPortMap:
	}
	return v.Text
}

// invariant: the eye reads a list's command as a hint after a gap
// (`daemon stopped  → station start`, §13.5); the pipe keeps the separator.
func join(items []page.Value, f func(page.Value) string) string {
	sep := token.Sep
	if tallies(items) {
		sep = Spaces(token.Gap)
	}
	var b strings.Builder
	for _, it := range items {
		s := f(it)
		if s == "" {
			continue
		}
		if b.Len() > 0 {
			if strings.HasPrefix(s, token.Arrow+" ") && it.Kind == page.KindCommand {
				b.WriteString(Spaces(token.Gap))
			} else {
				b.WriteString(sep)
			}
		}
		b.WriteString(s)
	}
	return b.String()
}

// value: a list of mark counts reads as a section title's counts do, `47 ●  27 ◐`.
func tallies(items []page.Value) bool {
	for _, it := range items {
		if it.Kind != page.KindCount || it.Unit != "" || Glyph(page.Mark(it.Label)) == "" {
			return false
		}
	}
	return len(items) > 0
}

func Numeric(k page.Kind) bool {
	switch k {
	case page.KindCount, page.KindSize, page.KindRatio, page.KindPercent, page.KindPort, page.KindPortMap, page.KindDuration, page.KindUntil:
		return true
	default:
	}
	return false
}

func Floor(k page.Kind) int {
	switch k {
	case page.KindVersion:
		return 8
	case page.KindURL:
		return 24
	case page.KindPath:
		return 12
	case page.KindText, page.KindList, page.KindIdentity:
		return 6
	default:
	}
	return 1 << 10
}

func Fit(v page.Value, width int) string {
	s := Show(v)
	if Width(s) <= width {
		return s
	}
	switch v.Kind {
	case page.KindPath:
		return path(s, width)
	case page.KindURL:
		return url(s, width)
	case page.KindList:
		return list(v.Items, width)
	case page.KindText, page.KindIdentity, page.KindVersion:
		return End(s, width)
	default:
	}
	return s
}

func Uncut(k page.Kind) bool {
	switch k {
	case page.KindPath, page.KindURL, page.KindList, page.KindText, page.KindIdentity, page.KindVersion, page.KindNone, "":
		return false
	default:
	}
	return true
}

func Least(v page.Value) int {
	// value: the narrowest width a path or URL is cut to while it still says
	// where it points: a path's first and last segments, a URL's floor.
	s := Show(v)
	switch v.Kind {
	case page.KindPath:
		if short, ok := shortPath(s); ok {
			return min(Width(s), Width(short))
		}
		return min(Width(s), Floor(page.KindPath))
	case page.KindURL:
		return min(Width(s), Floor(page.KindURL))
	default:
	}
	return Width(s)
}

func pathHead(s string) (head, last string, ok bool) {
	parts := strings.Split(s, "/")
	if len(parts) <= 3 {
		return "", "", false
	}
	head = parts[0]
	if head == "" {
		head = "/" + parts[1]
	}
	return head, parts[len(parts)-1], true
}

func shortPath(s string) (string, bool) {
	head, last, ok := pathHead(s)
	return head + "/" + token.Ellipsis + "/" + last, ok
}

func path(s string, width int) string {
	// invariant: a path keeps its first and last segments when they fit
	// (`/Applications/…/Tailscale`), else it is cut in the middle, its end
	// favored: `~/Library/La…tchman.plist` (§11.2).
	if short, ok := shortPath(s); ok && Width(short) <= width {
		return short
	}
	if width <= 2 {
		return End(s, width)
	}
	runes := []rune(s)
	keep := width - 1
	head := keep / 2
	return string(runes[:head]) + token.Ellipsis + string(runes[len(runes)-(keep-head):])
}

func url(s string, width int) string {
	scheme, rest, ok := strings.Cut(s, "://")
	if !ok {
		return Middle(s, width)
	}
	name, _, _ := strings.Cut(rest, "/")
	port := ""
	if i := strings.LastIndex(name, ":"); i >= 0 {
		name, port = name[:i], name[i:]
	}
	// invariant: only the host is cut: the scheme and the port are what a
	// reader needs to reach it (G4), so the label shortens before either goes.
	label, _, _ := strings.Cut(name, ".")
	fixed := scheme + "://" + token.Ellipsis + port
	if room := width - Width(fixed); room >= 1 {
		return scheme + "://" + fit(label, room) + token.Ellipsis + port
	}
	return End(scheme+"://"+label+token.Ellipsis+port, width)
}

func fit(s string, width int) string {
	runes := []rune(s)
	for len(runes) > 0 && Width(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func list(items []page.Value, width int) string {
	for n := len(items) - 1; n > 0; n-- {
		s := join(items[:n], Show) + " " + token.Ellipsis
		if Width(s) <= width {
			return s
		}
	}
	return End(join(items, Show), width)
}

func Bytes(n int64) string {
	// invariant: decimal units, as Finder, df -H and tart print sizes, so a size
	// station shows reads the same as the one the owner checks it against.
	const unit = 1000
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	value, suffix := float64(n)/unit, 0
	for value >= unit-0.05 && suffix < len(byteSuffixes)-1 {
		value /= unit
		suffix++
	}
	// value: three figures at most, as the vision prints sizes: 388 GB,
	// 73.3 GB, 5.8 GB.
	digits := 1
	if value >= 99.95 {
		digits = 0
	}
	return strconv.FormatFloat(value, 'f', digits, 64) + " " + string(byteSuffixes[suffix]) + "B"
}

const byteSuffixes = "KMGTPE"

type scale struct {
	div  float64
	unit string
}

func (s scale) of(n int64) string { return strconv.FormatFloat(float64(n)/s.div, 'f', 1, 64) }

func unitOf(total int64) (unit scale, text string) {
	s := scale{div: 1, unit: "B"}
	for i, div := 0, 1000.0; i < len(byteSuffixes) && float64(total) >= div; i, div = i+1, div*1000 {
		s = scale{div: div, unit: string(byteSuffixes[i]) + "B"}
	}
	return s, s.of(total) + " " + s.unit
}

func Duration(seconds int64) string {
	units := []struct {
		n    int64
		name string
	}{{86400, "d"}, {3600, "h"}, {60, "m"}, {1, "s"}}
	var parts []string
	rest := seconds
	for _, u := range units {
		if rest >= u.n && len(parts) < 2 {
			parts = append(parts, strconv.FormatInt(rest/u.n, 10)+u.name)
			rest %= u.n
		} else if len(parts) == 1 {
			break
		}
	}
	if len(parts) == 0 {
		return "0s"
	}
	return strings.Join(parts, " ")
}

func durationHead(seconds int64) int {
	d := Duration(seconds)
	if i := strings.IndexByte(d, ' '); i > 0 {
		return i
	}
	return len(d)
}
