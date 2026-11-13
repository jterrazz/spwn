package cli

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func parse(p catalog.Provider, outs []local.Output, late bool) accounts.Identity {
	if late {
		return accounts.Late(p)
	}
	if len(outs) == 0 {
		detail := "not installed"
		if p.Command != "" {
			detail = p.Command + " not on PATH"
		}
		return accounts.Absent(p.ID, accounts.ReasonMissing, detail)
	}
	switch p.ID {
	case catalog.ProviderGitHub:
		return parseGitHub(outs[0])
	case catalog.ProviderNPM:
		return parseNPM(outs[0])
	case catalog.ProviderClaude:
		return parseClaude(outs[0])
	case catalog.ProviderCloudflare:
		return parseWrangler(outs[0])
	case catalog.ProviderGit:
		return parseGit(outs)
	case catalog.ProviderSSH:
		return parseSSH(outs[0])
	case catalog.ProviderDocker:
		return parseDocker(outs[0])
	}
	return accounts.Absent(p.ID, accounts.ReasonMissing, "")
}

func lines(outs ...local.Output) []string {
	all := make([]string, 0, 4*len(outs))
	for _, o := range outs {
		all = append(all, strings.Split(o.Stdout, "\n")...)
		all = append(all, strings.Split(o.Stderr, "\n")...)
	}
	return all
}

var (
	ghAccount  = regexp.MustCompile(`Logged in to github\.com account (\S+) \(([^)]+)\)`)
	ghProtocol = regexp.MustCompile(`Git operations protocol: (\S+)`)
	ghScopes   = regexp.MustCompile(`Token scopes: (.*)$`)
	ghScope    = regexp.MustCompile(`'([^']*)'`)
)

func parseGitHub(out local.Output) accounts.Identity {
	acc := accounts.SignedIn(catalog.ProviderGitHub)
	acc.Source = "gh"
	var login, source, protocol string
	for _, line := range lines(out) {
		if m := ghAccount.FindStringSubmatch(line); m != nil && login == "" {
			login, source = m[1], m[2]
		}
		if m := ghProtocol.FindStringSubmatch(line); m != nil && protocol == "" {
			protocol = m[1]
		}
		if m := ghScopes.FindStringSubmatch(line); m != nil && acc.Scopes == nil {
			for _, s := range ghScope.FindAllStringSubmatch(m[1], -1) {
				acc.Scopes = append(acc.Scopes, s[1])
			}
		}
	}
	if login == "" {
		return accounts.Absent(catalog.ProviderGitHub, accounts.ReasonUnset, "not logged in")
	}
	acc.Who = []string{login}
	acc.Notes = slices.DeleteFunc([]string{source, protocol}, func(s string) bool { return s == "" })
	return acc
}

func parseNPM(out local.Output) accounts.Identity {
	switch {
	case strings.Contains(out.Stderr, "E401") || strings.Contains(out.Stderr, "E403"):
		return accounts.Broken(catalog.ProviderNPM, accounts.ReasonRefused, "token refused by registry.npmjs.org")
	case strings.Contains(out.Stderr, "ENEEDAUTH") || out.ExitCode != 0:
		return accounts.Absent(catalog.ProviderNPM, accounts.ReasonUnset, "not logged in")
	}
	login := strings.TrimSpace(strings.SplitN(strings.TrimSpace(out.Stdout), "\n", 2)[0])
	if login == "" {
		return accounts.Absent(catalog.ProviderNPM, accounts.ReasonUnset, "not logged in")
	}
	acc := accounts.SignedIn(catalog.ProviderNPM)
	acc.Who = []string{login}
	return acc
}

func parseClaude(out local.Output) accounts.Identity {
	var status struct {
		LoggedIn         bool   `json:"loggedIn"`
		AuthMethod       string `json:"authMethod"`
		Email            string `json:"email"`
		SubscriptionType string `json:"subscriptionType"`
	}
	if json.Unmarshal([]byte(out.Stdout), &status) != nil || !status.LoggedIn {
		return accounts.Absent(catalog.ProviderClaude, accounts.ReasonUnset, "not logged in")
	}
	acc := accounts.SignedIn(catalog.ProviderClaude)
	acc.Who = []string{status.Email}
	if status.Email == "" {
		acc.Who = []string{accounts.Unknown}
	}
	if status.SubscriptionType != "" {
		acc.Scopes = []string{status.SubscriptionType}
	}
	acc.Detail = status.AuthMethod
	return acc
}

var (
	wranglerEmail = regexp.MustCompile(`associated with the email (\S+?)\.?\s*$`)
	wranglerRow   = regexp.MustCompile(`^[\x{2502}|]\s*(.+?)\s*[\x{2502}|]\s*[0-9a-f]{32}\s*[\x{2502}|]`)
)

func parseWrangler(out local.Output) accounts.Identity {
	all := lines(out)
	acc := accounts.SignedIn(catalog.ProviderCloudflare)
	acc.Source = "wrangler"
	for _, line := range all {
		if strings.Contains(line, "You are not authenticated") {
			return accounts.Absent(catalog.ProviderCloudflare, accounts.ReasonUnset, "not logged in")
		}
		if m := wranglerEmail.FindStringSubmatch(line); m != nil && acc.Who == nil {
			acc.Who = []string{m[1]}
		}
		if m := wranglerRow.FindStringSubmatch(line); m != nil {
			acc.Scopes = append(acc.Scopes, m[1])
		}
	}
	if acc.Who == nil {
		return accounts.Absent(catalog.ProviderCloudflare, accounts.ReasonUnset, "not logged in")
	}
	return acc
}

func parseGit(outs []local.Output) accounts.Identity {
	var values []string
	for _, o := range outs {
		if v := strings.TrimSpace(strings.SplitN(o.Stdout, "\n", 2)[0]); o.ExitCode == 0 && v != "" {
			values = append(values, v)
		}
	}
	if len(values) == 0 {
		return accounts.Absent(catalog.ProviderGit, accounts.ReasonUnset, "no user.name nor user.email")
	}
	acc := accounts.SignedIn(catalog.ProviderGit)
	acc.Who = values
	return acc
}

func parseSSH(out local.Output) accounts.Identity {
	switch out.ExitCode {
	case 0:
	case 1:
		return accounts.Absent(catalog.ProviderSSH, accounts.ReasonUnset, "no key in the agent")
	default:
		return accounts.Absent(catalog.ProviderSSH, accounts.ReasonUnset, "no agent")
	}
	var comments []string
	n := 0
	for _, line := range strings.Split(out.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.HasPrefix(fields[1], "SHA256:") {
			continue
		}
		n++
		comment := strings.Join(fields[2:], " ")
		if i := strings.LastIndex(comment, " ("); i >= 0 {
			comment = comment[:i]
		}
		if comment != "" && !slices.Contains(comments, comment) {
			comments = append(comments, comment)
		}
	}
	if n == 0 {
		return accounts.Absent(catalog.ProviderSSH, accounts.ReasonUnset, "no key in the agent")
	}
	acc := accounts.SignedIn(catalog.ProviderSSH)
	acc.Who = []string{keys(n)}
	acc.Scopes = comments
	return acc
}

func keys(n int) string {
	if n == 1 {
		return "1 key"
	}
	return strconv.Itoa(n) + " keys"
}

func dockerStore(body []byte) (store string, listed bool) {
	var c struct {
		Auths       map[string]json.RawMessage `json:"auths"`
		CredsStore  string                     `json:"credsStore"`
		CredHelpers map[string]string          `json:"credHelpers"`
	}
	if json.Unmarshal(body, &c) != nil {
		return "", false
	}
	if c.CredsStore == "" {
		for _, h := range c.CredHelpers {
			c.CredsStore = h
			break
		}
	}
	return c.CredsStore, len(c.Auths) > 0 || c.CredsStore != ""
}

func parseDocker(out local.Output) accounts.Identity {
	var registries map[string]string
	if out.ExitCode != 0 || json.Unmarshal([]byte(out.Stdout), &registries) != nil {
		return accounts.Broken(catalog.ProviderDocker, accounts.ReasonFailing, "the credential helper did not list its registries")
	}
	if len(registries) == 0 {
		return accounts.Absent(catalog.ProviderDocker, accounts.ReasonUnset, "no registry login")
	}
	names := make([]string, 0, len(registries))
	for r := range registries {
		names = append(names, r)
	}
	slices.Sort(names)
	acc := accounts.SignedIn(catalog.ProviderDocker)
	for _, r := range names {
		domain := strings.TrimPrefix(strings.TrimPrefix(r, "https://"), "http://")
		domain, _, _ = strings.Cut(domain, "/")
		if !slices.Contains(acc.Who, registries[r]) {
			acc.Who = append(acc.Who, registries[r])
		}
		acc.Scopes = append(acc.Scopes, domain)
	}
	return acc
}
