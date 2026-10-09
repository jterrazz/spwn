package google

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// WizardWalkthrough is the user-facing instructions printed when
// no client.json exists yet. Kept here (vs the CLI package) so the
// language stays consistent across callers — the same text gets
// printed by the dashboard and by `spwn auth login google` itself.
const WizardWalkthrough = `Google requires a one-time GCP project setup (Google policy — they don't allow
shared OAuth clients for sensitive scopes). 5–10 minutes, free, no credit card.

  1. Open https://console.cloud.google.com/projectcreate
     → name the project "spwn" (or anything), Create.

  2. Enable the APIs you want spwn to use:
       Gmail API:    https://console.cloud.google.com/apis/library/gmail.googleapis.com
       Calendar API: https://console.cloud.google.com/apis/library/calendar-json.googleapis.com
     → Click "Enable" on each.

  3. Configure the OAuth consent screen:
       https://console.cloud.google.com/apis/credentials/consent
     → User type: External
     → App name: "spwn" (or anything), your email, Save & Continue
     → Scopes: skip (we'll request at runtime)
     → Test users: add your own Google email — REQUIRED, otherwise login fails
     → Save & Continue

  4. Create the OAuth client:
       https://console.cloud.google.com/apis/credentials
     → Create credentials → OAuth client ID
     → Application type: "Desktop app"
     → Name: "spwn"
     → Create
     → Copy the client_id (looks like 123456789-xxxxx.apps.googleusercontent.com)
     → Copy the client_secret if shown (Desktop apps may or may not, both work)

  5. Paste them below. They're stored at ~/.spwn/credentials/google/client.json
     and never leave your machine.`

// PromptClient walks the user through the GCP setup and captures
// their OAuth client_id / client_secret. Reads from in, prints to
// out — both injectable for tests.
func PromptClient(in io.Reader, out io.Writer) (*ClientConfig, error) {
	fmt.Fprintln(out, WizardWalkthrough)
	fmt.Fprintln(out)

	r := bufio.NewReader(in)

	clientID, err := promptLine(r, out, "Client ID", true)
	if err != nil {
		return nil, err
	}
	clientSecret, err := promptLine(r, out, "Client Secret (leave blank if Desktop app didn't show one)", false)
	if err != nil {
		return nil, err
	}

	c := &ClientConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       DefaultScopes,
	}
	return c, nil
}

func promptLine(r *bufio.Reader, w io.Writer, label string, required bool) (string, error) {
	for {
		fmt.Fprintf(w, "%s: ", label)
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line != "" || !required {
			return line, nil
		}
		fmt.Fprintln(w, "(required, please enter a value)")
	}
}
