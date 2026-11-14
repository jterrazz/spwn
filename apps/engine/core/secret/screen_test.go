package secret_test

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

// value: every issuer literal is built by join, so no file of the
// repository holds a string a push protection reads as a token.
func token(prefix, body string) string { return prefix + body }

func TestEachShapeIsWithheldAndTheWordsAroundIt(t *testing.T) {
	for _, c := range []struct{ shape, text, want string }{
		{"GitHub", "token " + token("g", "hp_A1b2C3") + " set", "token withheld set"},
		{"GitHub fine-grained", token("github"+"_pat_", "11ABCDEF_xyz"), "withheld"},
		{"Anthropic", "key=" + token("sk-"+"ant-", "api03-abc"), "key=withheld"},
		{"an sk- key", token("sk"+"-", "proj-"+strings.Repeat("x", 16)), "withheld"},
		{"a package registry token", token("n", "pm_A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8"), "withheld"},
		{"a Stripe key", token("sk"+"_live_", "A1b2C3d4E5f6G7h8I9j0"), "withheld"},
		{"a Stripe restricted key", token("rk"+"_test_", "A1b2C3d4E5f6G7h8I9j0"), "withheld"},
		{"Slack", token("xo"+"xb-", "1234-5678"), "withheld"},
		{"a Slack session token", token("xo"+"xs-", "1234-5678"), "withheld"},
		{"a Slack app token", token("xa"+"pp-", "1-A0123-456-abcdef"), "withheld"},
		{"a JWT", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.c2lnbmF0dXJl", "withheld"},
		{"an AWS key", "id " + token("AK"+"IA", "ABCDEFGHIJKLMNOP") + ".", "id withheld."},
		{"a private key", "-----BEGIN RSA " + "PRIVATE KEY-----\nabc\n-----END RSA " + "PRIVATE KEY-----\nafter", "withheld\nafter"},
		{"a URL's userinfo", "fetch refused for https://bot:hunter2@github.com/o/r", "fetch refused for https://withheld@github.com/o/r"},
		{"a password holding a slash", "https://bot:pa/ss@host/x", "https://withheld@host/x"},
		{"a token as a URL's user", "clone https://" + token("a1b2c3d4", "e5f6a7b8c9d0e1f2") + "@github.com/o/r", "clone https://withheld@github.com/o/r"},
		{"a token query", "GET /hook?id=4&token=hunter2 200", "GET /hook?id=4&token=withheld 200"},
		{"the query's other secret names", "GET /x?api_key=hunter2&client_secret=s3&sig=abc&X-Amz-Signature=f00&page=2 200", "GET /x?api_key=withheld&client_secret=withheld&sig=withheld&X-Amz-Signature=withheld&page=2 200"},
		{"an Authorization header", "Authorization: Bearer hunter2 sent", "Authorization: Bearer withheld sent"},
		{"an Authorization header in JSON", `{"Authorization": "Bearer hunter2"}`, `{"Authorization": "Bearer withheld"}`},
		{"a token in a URL's path", "post https://hooks.slack.com/services/T0AB12CD3/B0EF45GH6/" + token("a1B2c3D4", "e5F6g7H8i9J0k1L2"), "post https://hooks.slack.withheld"},
		{"a run after a dot", "tok." + token("a1B2c3D4e5F6g7H8", "i9J0k1L2m3N4o5P6q7R8"), "tok.withheld"},
		{"a UUID", "machine 123e4567-e89b-12d3-a456-426614174000 gone", "machine withheld gone"},
		{"a bare base64 run", "pass QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo0MTIz now", "pass withheld now"},
		{"a 32 hex run", "0123456789abcdef0123456789abcdef", "withheld"},
		{"a 40 hex run in capitals", strings.Repeat("A1B2C3D4E5", 4), "withheld"},
		{"a 41 hex run", strings.Repeat("a1", 20) + "b", "withheld"},
		{"a 63 hex run", strings.Repeat("a1", 31) + "b", "withheld"},
		{"a mixed base64url run of short segments", "set Ab3_dE5-fG7h_Ij9-kL1m_No3-pQ5r_St7-uV9w_Xy1 now", "set withheld now"},
		{"a mixed base64 run of 32", "pass Ab3+dE5/fG7h+Ij9/kL1m+No3/pQ5r+S now", "pass withheld now"},
		{"a mixed base64 run that starts at a slash", "pass /Ab3dE5fG7hIj9kL1mNo3pQ5rSt7uV9wXy1 now", "pass withheld now"},
		{"a slash-led base64 run holding a plus", "pass /Ab3dE5fG7h/Ij9kL1mNo3+pQ5rSt7uV9wXy1 now", "pass withheld now"},
		{"a GitLab token", token("gl"+"pat-", "A1b2C3d4E5f6G7h8I9j0"), "withheld"},
		{"a registry access token", token("dc"+"kr_pat_", "A1b2C3d4E5f6G7h8I9j0K1l2M3n"), "withheld"},
		{"a DigitalOcean token", token("do"+"p_v1_", strings.Repeat("0a1b2c3d", 8)), "withheld"},
		{"a Google API key", token("AI"+"za", "SyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6q"), "withheld"},
		{"a query named for a key in one word", "GET /x?apikey=hunter2&accessToken=t0k 200", "GET /x?apikey=withheld&accessToken=withheld 200"},
		{"a password holding an at", "https://bob:p@ssw0rd@host/x", "https://withheld@host/x"},
		{"a password holding an at and a slash", "https://bob:p@ss/w0rd@host/x", "https://withheld@host/x"},
		{"a password before a scope's path", "https://bot:hunter2@registry.example/@scope/pkg", "https://withheld@registry.example/@scope/pkg"},
		{"a query named for a token in two words", "GET /x?X-Amz-Security-Token=abc&id_token=t&keys=a 200", "GET /x?X-Amz-Security-Token=withheld&id_token=withheld&keys=a 200"},
		{"a token as a URL path's segment", "get https://github.com/A1b2C3d4E5f6G7h8I9j0K1l2M3/x", "get https://github.withheld"},
		{"a token split in a URL's path", "get https://github.com/A1b2C3d4/E5f6G7h8/I9j0K1l2/x", "get https://github.withheld"},
		{"a token as long as a test's name", "--- FAIL: Ab3dE5fG7hIj9kL1mNo3pQ5rSt7uV9wXy1zA2bC (0.01s)", "--- FAIL: withheld (0.01s)"},
		{"a password holding a quote", "https://bob:it'sAp4ss@host/x", "https://withheld@host/x"},
		{"a password holding angle brackets", "https://bob:pa<ss>w0rd@host/x", "https://withheld@host/x"},
		{"a password holding a double quote", `https://bob:pa"ss@host/x`, "https://withheld@host/x"},
		{"a password ending in a slash", "https://bob:s3cr3tpass/@host/x", "https://withheld@host/x"},
		{"a password in a JSON value", `{"url":"https://bob:hunter2@host"}`, `{"url":"https://withheld@host"}`},
		{"a query named for a secret after any word", "GET /u?app_secret=hunter2hunter2&webhook_secret=Sh0rt&bot_token=1234&hub.verify_token=t0k 200", "GET /u?app_secret=withheld&webhook_secret=withheld&bot_token=withheld&hub.verify_token=withheld 200"},
		{"a query named for a signature or a token after any word", "GET /u?oauth_signature=tR3%2BTy81&oauth_token=0123abcd&secrets=abc123 200", "GET /u?oauth_signature=withheld&oauth_token=withheld&secrets=withheld 200"},
		{"a query named for a key that is a secret", "GET /u?master_key=abc123&access_key=abc123&key=AIza1 200", "GET /u?master_key=withheld&access_key=withheld&key=withheld 200"},
		{"a token in a URL's query", "https://h.example/x?s=AbCdEfGhIj/KlMnOpQrSt/q1w2e3r4t5y6", "https://h.example/x?withheld"},
		{"a token in a URL's fragment", "https://h.example/cb#t=AbCdEfGhIjKl/MnOpQrStWx/w1x2y3z4a5b6", "https://h.example/cb#withheld"},
		{"a token after a scheme no URL has", "see x://AbCdEfGhIjKlMn/q1w2e3r4t5y6u7/ZyXwVuTsRqPoNm", "see x:withheld"},
		{"a token as a lower case segment of a URL's path", "https://app.example.com/Account/Reset/k3j4h5g6f7d8s9a0q1w", "https://app.example.withheld"},
		{"a capitalised run of digits", "pass Sk918273645091827364509182736450 now", "pass withheld now"},
		{"a passphrase of capitalised words", "pass CorrectHorseBatteryStapleZebraLimeTango2024 now", "pass withheld now"},
		{"a capitalised run with a digit", "pass XkcdPasswordAbcDefGhiJklMnoPqrStu9 now", "pass withheld now"},
		{"a password of digits before a quote that closes no value", `https://bob:1234"x@host/x`, "https://withheld@host/x"},
		{"a password holding a comma", "https://bob:pa,ss@host/x", "https://withheld@host/x"},
		{"a password holding a quote before a comma", `https://bob:pa",ss@host/x`, "https://withheld@host/x"},
		{"a password in compact JSON before an email", `{"url":"https://bob:hunter2@host","email":"a@b.com"}`, `{"url":"https://withheld@b.com"}`},
	} {
		got, withheld := secret.Screen(c.text)
		if got != c.want || !withheld {
			t.Errorf("%s: Screen(%q) = %q, %v; want %q, true", c.shape, c.text, got, withheld, c.want)
		}
		if !secret.Shaped(c.text) {
			t.Errorf("%s: %q is not Shaped", c.shape, c.text)
		}
	}
}

func TestTheMachinesOwnDataIsShown(t *testing.T) {
	hex40, hex64 := strings.Repeat("0a1b2c3d4e", 4), strings.Repeat("0a1b2c3d", 8)
	for _, text := range []string{
		"/Users/jterrazz/Developer/0123456789abcdef0123456789abcdef/x",
		"~/" + strings.Repeat("ab12", 9),
		"station v3.1.0-144-g833ff9985 (" + hex40 + ")",
		"ghcr.io/cirruslabs/macos@sha256:" + strings.Repeat("A1b2", 16),
		"commit " + hex40,
		"digest " + hex64,
		"SHA256:76MCcfWNZv0Ixcc4fM0b6Ue1o8l7W1Xo0n3zKhK1Vy4",
		"specs/usecases/machinefile/apply/a-failing-line-withholds",
		"admin@jterrazz.com, read:org, git@github.com:o/r.git",
		"Jean-Baptiste Terrazzoni",
		"machine station-gate-cold-2026-11-13-redaction not found",
		"refs/heads/feat/redaction-pre-verify1-2026-11-13",
		"specs/usecases/machinefile/apply/a-failing-line-withholds-its-token.spec.yaml",
		"ending:jterrazz-macbook-pro/linux-1/agent-code-2026-11",
		"short/macbook-pro/linux-1/review-bot",
		token("n", "pm_config_cache") + "=/Users/x/.cache",
		"ssh://git@github.com/o/r.git",
		"http://localhost:4873/@jterrazz/x-2026-11-13-release-candidate",
		"/Users/jterrazz/.cache/go-build/0a/" + hex64 + "-d",
		"/private/tmp/run/1f4e8ae8-830c-45f6-8826-69e96d7c385b/scratchpad",
		"GET /x?author=jb&page=2",
		"GET /x?keyboard=qwerty&monkey=banana&turkey=1",
		"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT admin@x",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl admin@jterrazz.com",
		"\"integrity\": \"sha512-z4PhNX7vuL3xVChQ1m2AB9Yg5AULVxXcg/SpIdNs6c5H0NE8XYXysP+DGNKHfuwvY7kxvUdBeoGlODJ6+SfaPg==\"",
		"sha1-Ql5HUkRp9bSsBk0WT7Kn2ZtdT0M=",
		"file:///Users/jterrazz/.cache/0123456789abcdef0123456789abcdef",
		"/Users/jterrazz/Library/Caches/ABCDEFAbcdef0123456789ABCDEFabcdef0123",
		"GET /x?cache_key=v1&sort_key=name&public_key=AAAA&keys=a,b",
		"open https://github.com/AcmeLabs/widgets/releases/tag/4.4.0",
		"open https://github.com/Homebrew/homebrew-core/commit/" + hex40,
		"curl -fsSL https://github.com/NixOS/nix/releases/download/2.24.9/nix-2.24.9.tar.xz",
		"--- FAIL: TestTripwireCatchesAnAnswerWith2Secrets (0.01s)",
		"--- FAIL: BenchmarkScreenOfAnErrorWithTwentyWords2 (0.01s)",
		`{"remote":"https://github.com/o/r","email":"a@b.com"}`,
		"https://github.com/o/r/@scope/pkg",
		"GET /x?designs=a&cwd=/tmp&monkeys=3&keyword=go&sorting=asc",
		"clone git://github.com/AcmeLabs/widgets/tree/release-4",
		"open https://github.com/AcmeLabs/widgets/blob/v2/docs/install-10.md",
		`{"url":"https://example.com","email":"a@b.com"}`,
		`{"url":"https://example.com:8080","email":"a@b.com"}`,
		`"homepage":"https://x.dev","devDependencies":{"@types/node":"^20"}`,
		`"homepage":"https://x.dev:8080/docs","devDependencies":{"@types/node":"^20"}`,
		"https://x.io,2024-01-01T10:00:00,me@x.io",
		`{'url':'https://x.dev','owner':'a@b.com'}`,
		`<a href="https://x.dev:8080">a@b.com</a>`,
		"http://[fe80::1]:80/a@b",
	} {
		if got, withheld := secret.Screen(text); got != text || withheld {
			t.Errorf("Screen(%q) = %q, %v: the machine's own data was withheld", text, got, withheld)
		}
	}
}

func TestAScreenKeepsItsLines(t *testing.T) {
	text := "line one\n\tkey " + token("g", "ho_abc") + "\nline three"
	if got, _ := secret.Screen(text); got != "line one\n\tkey withheld\nline three" {
		t.Errorf("Screen moved the lines: %q", got)
	}
}

func TestIssuedIsTheShapesThatNameTheirIssuer(t *testing.T) {
	for _, text := range []string{token("g", "hs_abc"), token("xo"+"xp-", "1-2"), "x -----BEGIN OPENSSH " + "PRIVATE KEY----- y"} {
		if !secret.Issued(text) {
			t.Errorf("%q names its issuer, Issued said no", text)
		}
	}
	for _, text := range []string{token("sk"+"_live_", "A1b2C3d4E5f6G7h8I9j0"), token("xa"+"pp-", "1-A0"), token("xo"+"xo-", "1-2"), token("gl"+"pat-", "A1b2C3d4E5f6G7h8I9j0"), token("dc"+"kr_pat_", "A1b2C3d4E5f6G7h8I9j0K1l2M3n"), token("do"+"p_v1_", strings.Repeat("0a1b2c3d", 8)), token("AI"+"za", "SyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6q")} {
		if !secret.Issued(text) {
			t.Errorf("%q names its issuer, Issued said no", text)
		}
	}
	for _, text := range []string{"QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo0MTIz", "https://bot:hunter2@host/", "disk-usage", token("n", "pm_config_cache"), "123e4567-e89b-12d3-a456-426614174000"} {
		if secret.Issued(text) {
			t.Errorf("%q names no issuer, Issued said yes", text)
		}
	}
}

func TestKeptIsScreensWordsWithoutTheFlag(t *testing.T) {
	text := "key " + token("g", "ho_abc") + " after"
	want, _ := secret.Screen(text)
	if got := secret.Kept(text); got != want || got == text {
		t.Errorf("Kept(%q) = %q, want %q", text, got, want)
	}
}

func compactJSONOfURLs(n int) string {
	var b strings.Builder
	b.WriteString(`{"links":[`)
	for range n {
		b.WriteString(`{"url":"https://x.dev/a","by":"a@b.com"},`)
	}
	b.WriteString(`{}]}`)
	return b.String()
}

func TestAWordOfManyURLsIsShownWhole(t *testing.T) {
	text := compactJSONOfURLs(20000)
	if got := secret.Kept(text); got != text {
		t.Errorf("a compact JSON of %d bytes was changed by its screen", len(text))
	}
}

func BenchmarkKeptOfAWordOfManyURLs(b *testing.B) {
	text := compactJSONOfURLs(20000)
	b.SetBytes(int64(len(text)))
	for b.Loop() {
		secret.Kept(text)
	}
}
