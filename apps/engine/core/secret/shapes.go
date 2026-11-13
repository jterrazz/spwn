package secret

import (
	"regexp"
	"strconv"
	"strings"
)

const Marker = "withheld" // value: the word a withheld secret reads as, on every surface.

// value: a pure lowercase hex segment of exactly these lengths is a git
// object or a sha256 and is shown; every other long segment of letters and
// digits is withheld.
func shownHex() []int { return []int{40, 64} }

// value: a run this long, of these characters, that mixes upper case, lower
// case and digits is a token whatever its segments; any other run is looked
// at segment by segment, its segments apart by a separator, and a segment
// this long holding a letter and a digit is a token. Neither is one in a
// path from a root, after a named digest or an integrity's algorithm, as a
// public key's body, or as shown hex. A slug, a git ref, a relative path or
// a VM's name is lower case in short segments, and is shown.
const (
	runMin     = 32
	runClass   = `[A-Za-z0-9_+/=-]`
	segmentMin = 20
	separators = "-_/"
)

// value: a segment of a URL's path this long that holds a letter and a digit
// is no word of the path.
const urlSegmentMin = 16

// value: the characters a word of a path from a root starts after.
const wordEdges = " \t\r\n\"'`([{<,;=:"

// value: the shapes that name their issuer (GitHub, GitLab, Anthropic and
// any `sk-` key, Stripe, npm, a container registry, DigitalOcean, Slack, a
// Google API key, a JWT, an AWS access key); none is ever the machine's own
// data. The npm and registry prefixes are spelt with a class so the word
// reads as the token's prefix, not as the tool.
func issuers() []string {
	return []string{
		`\bgh[opusr]_[A-Za-z0-9_]+`,
		`\bgithub_pat_[A-Za-z0-9_]+`,
		`\bsk-ant-[A-Za-z0-9_-]+`,
		`\bsk-[A-Za-z0-9_-]{16,}`,
		`\b[sr]k_(?:live|test)_[A-Za-z0-9]{16,}`,
		`\b[n]pm_[A-Za-z0-9]{36}\b`,
		`\bglpat-[A-Za-z0-9_-]{20,}`,
		`\b[d]ckr_pat_[A-Za-z0-9_-]{20,}`,
		`\bdop_v1_[a-f0-9]{64}\b`,
		`\bAIza[0-9A-Za-z_-]{35}`,
		`\bxox[abposr]-[A-Za-z0-9-]+`,
		`\bxapp-[A-Za-z0-9-]+`,
		`\beyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`,
		`\bAKIA[0-9A-Z]{16}\b`,
		`-----BEGIN [A-Z ]*PRIVATE KEY-----`,
	}
}

var (
	issued     = regexp.MustCompile(strings.Join(issuers(), "|"))
	privateKey = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----(?s:.*?)(?:-----END [A-Z ]*PRIVATE KEY-----|\z)`)
	header     = regexp.MustCompile(`(?i)(authorization["']?\s*[:=]\s*["']?(?:bearer|basic|token)\s+)[^\s"',]+`)
	port       = regexp.MustCompile(`^\d+/`)
	query      = regexp.MustCompile(`(?i)([?&](?:[\w.-]*(?:token|secret|password|passwd|pwd|signature|sig|credential)s?|(?:x-(?:amz|goog)-)?(?:(?:api|access|auth|client|encryption|id|master|private|refresh|secret|security|session|signing)[_.-]?keys?|key))=)[^&\s#"']+`)
	identifier = regexp.MustCompile(`^(?:Test|Benchmark|Example|Fuzz)(?:[A-Z][a-z]+)+(?:\d{1,4}(?:[A-Z][a-z]+)*)?$`)
	run        = regexp.MustCompile(runClass + `{` + strconv.Itoa(runMin) + `,}`)
	uuid       = regexp.MustCompile(`[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}`)
	digest     = regexp.MustCompile(`(?i)sha(?:1|256|384|512):$`)
	integrity  = regexp.MustCompile(`^sha(?:1|256|384|512)-`)
	scheme     = regexp.MustCompile(`(?i)(?:^|[^a-z])(?:https?|git|[Ss]sh|file)$`)
	publicKey  = regexp.MustCompile(`(?:ssh-(?:ed25519|rsa|dss)|ecdsa-sha2-nistp(?:256|384|521)) $`)
)

func Issued(text string) bool {
	// value: whether text holds a shape that names its issuer: the set no
	// fixture, frame or page may ever carry, since none collides with data.
	return issued.MatchString(text)
}

func credentials(text string) string {
	// invariant: a URL's userinfo runs from `://` to the word's last `@` that
	// does not follow a `/` (its last `@` when each one does), whatever it
	// holds short of whitespace, so a password holding an `@`, a quote, a `<`
	// or a `/`, or ending in one, is withheld whole. It is withheld when it
	// holds a password, or when its user alone is token-shaped; a user holds
	// no `/`, and a host and its port before a path (`localhost:4873/@scope`)
	// is no userinfo.
	var b strings.Builder
	last := 0
	for from := 0; ; {
		i := strings.Index(text[from:], "://")
		if i < 0 {
			break
		}
		start := from + i + len("://")
		end := len(text)
		if j := strings.IndexAny(text[start:], " \t\r\n"); j >= 0 {
			end = start + j
		}
		at := lastAt(text[start:end])
		from = start
		if at < 0 || !userinfo(text[start:start+at]) {
			continue
		}
		b.WriteString(text[last:start])
		b.WriteString(Marker)
		last, from = start+at, start+at
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

func lastAt(word string) int {
	last, free := -1, -1
	for i := range len(word) {
		if word[i] != '@' {
			continue
		}
		last = i
		if i > 0 && word[i-1] != '/' {
			free = i
		}
	}
	if free >= 0 {
		return free
	}
	return last
}

func userinfo(info string) bool {
	user, password, _ := strings.Cut(info, ":")
	switch {
	case strings.Contains(user, "/"):
		return false
	case password != "":
		return !port.MatchString(password)
	}
	return tokenUser(user)
}

func tokenUser(user string) bool {
	return len(user) >= 16 && hasLetter(user) && strings.ContainsAny(user, "0123456789")
}

func bare(text string, at []int) bool {
	r := text[at[0]:at[1]]
	switch {
	case digest.MatchString(text[:at[0]]), integrity.MatchString(r):
		return false
	case publicKey.MatchString(text[:at[0]]) && strings.HasPrefix(r, "AAAA"):
		return false
	case identifier.MatchString(r), urlWords(text, at):
		return false
	case mixed(r) && !inPath(text, at[0]):
		return true
	}
	for _, m := range uuid.FindAllStringIndex(r, -1) {
		if !inPath(text, at[0]+m[0]) {
			return true
		}
	}
	start := 0
	for i := 0; i <= len(r); i++ {
		if i < len(r) && !strings.ContainsRune(separators, rune(r[i])) {
			continue
		}
		if tokenSegment(r[start:i]) && !inPath(text, at[0]+start) {
			return true
		}
		start = i + 1
	}
	return false
}

func tokenSegment(s string) bool {
	return len(s) >= segmentMin && hasLetter(s) && strings.ContainsAny(s, "0123456789") && !shown(s)
}

func mixed(r string) bool {
	return strings.ContainsAny(r, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") && strings.ContainsAny(r, "abcdefghijklmnopqrstuvwxyz") && strings.ContainsAny(r, "0123456789")
}

func inPath(text string, at int) bool {
	// invariant: a path from a root is a word that starts at `/` or `~`
	// (not after a `:`, as in a URL's `://`, except a `file:` URL's) and
	// holds two segments and more of a path's characters, so a base64
	// secret that starts at a `/` is no path.
	start := strings.LastIndexAny(text[:at], wordEdges) + 1
	end := len(text)
	if i := strings.IndexAny(text[start:], wordEdges); i >= 0 {
		end = start + i
	}
	word := text[start:end]
	switch {
	case !strings.HasPrefix(word, "/") && !strings.HasPrefix(word, "~"):
		return false
	case start > 0 && text[start-1] == ':' && !strings.HasSuffix(text[:start], "file:"):
		return false
	case strings.ContainsAny(word, "+="):
		return false
	}
	segments := 0
	for _, s := range strings.Split(word, "/") {
		if s != "" {
			segments++
		}
	}
	return segments >= 2
}

func urlWords(text string, at []int) bool {
	// invariant: a run in a URL's path (after an http, https, git, ssh or
	// file scheme's `://`, before any `?` or `#`) is no token when it holds
	// two segments and more between `/` and each is a word: shorter than a
	// token's segment, not mixing upper case with digits, and shorter than a
	// URL's segment when it holds a letter and a digit, or shown hex. A
	// segment like a webhook's `T0AB12CD3` is no word; a query's value and a
	// fragment are read by the other shapes.
	start := strings.LastIndexAny(text[:at[0]], wordEdges) + 1
	if start == 0 || text[start-1] != ':' || !strings.HasPrefix(text[start:], "//") || !webScheme(text[:start-1]) || strings.ContainsAny(text[start:at[0]], "?#") {
		return false
	}
	segments := strings.Split(strings.Trim(text[at[0]:at[1]], "/"), "/")
	if len(segments) < 2 {
		return false
	}
	for _, s := range segments {
		upper, digit := strings.ContainsAny(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"), strings.ContainsAny(s, "0123456789")
		if !shown(s) && (s == "" || len(s) >= segmentMin || upper && digit || digit && hasLetter(s) && len(s) >= urlSegmentMin) {
			return false
		}
	}
	return true
}

func webScheme(before string) bool {
	return scheme.MatchString(before)
}

func shown(r string) bool {
	for _, n := range shownHex() {
		if len(r) == n && strings.Trim(r, "0123456789abcdef") == "" {
			return true
		}
	}
	return false
}

func hasLetter(r string) bool {
	for _, c := range r {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}
