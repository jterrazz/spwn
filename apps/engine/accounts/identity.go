package accounts

import (
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	ReasonUnset       = "unset"
	ReasonMissing     = "missing"
	ReasonRefused     = "refused"
	ReasonLate        = "late"
	ReasonFailing     = "failing"
	ReasonExpiring    = "expiring"
	ReasonUnreachable = "unreachable"

	Unknown = "?"
)

type Identity struct {
	Provider string
	State    catalog.State
	Reason   string
	Who      []string
	Scopes   []string
	Source   string
	Detail   string
	Notes    []string
}

func SignedIn(provider string) Identity {
	return Identity{Provider: provider, State: catalog.StateOK}
}

func Absent(provider, reason, detail string) Identity {
	return Identity{Provider: provider, State: catalog.StateAbsent, Reason: reason, Detail: detail}
}

func Broken(provider, reason, detail string) Identity {
	return Identity{Provider: provider, State: catalog.StateBroken, Reason: reason, Detail: detail, Who: []string{Unknown}}
}

func Late(p catalog.Provider) Identity {
	return Identity{Provider: p.ID, State: catalog.StateDegraded, Reason: ReasonLate, Who: []string{Unknown}, Detail: "no answer in " + strconv.Itoa(int(p.Budget.Seconds())) + "s"}
}

func Fingerprint(line string) (fingerprint, comment string, ok bool) {
	// value: the SHA256 fingerprint and the comment of a public key line, found
	// by its base64 blob wherever it stands in the line.
	fields := strings.Fields(line)
	for i, f := range fields {
		blob, err := base64.StdEncoding.DecodeString(f)
		if err != nil || len(blob) < 16 || i == 0 && !strings.HasPrefix(f, "AAAA") {
			continue
		}
		sum := sha256.Sum256(blob)
		return "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(sum[:]), "="), strings.Join(fields[i+1:], " "), true
	}
	return "", "", false
}
