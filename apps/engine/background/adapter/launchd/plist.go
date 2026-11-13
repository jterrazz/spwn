package launchd

import (
	"bytes"
	"encoding/xml"
	"sort"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
)

func plist(spec contract.JobSpec) string {
	keys := make([]string, 0, len(spec.Env))
	for k := range spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	b.WriteString("\t<key>Label</key>\n\t<string>" + xmlText(spec.Label) + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range spec.Program {
		b.WriteString("\t\t<string>" + xmlText(arg) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
	for _, k := range keys {
		b.WriteString("\t\t<key>" + xmlText(k) + "</key>\n\t\t<string>" + xmlText(spec.Env[k]) + "</string>\n")
	}
	b.WriteString("\t</dict>\n")
	if spec.AtLogin {
		b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	}
	// invariant: a job kept alive is restarted after any exit, and the
	// processes it started outlive it.
	if spec.KeepAlive {
		b.WriteString("\t<key>KeepAlive</key>\n\t<true/>\n\t<key>AbandonProcessGroup</key>\n\t<true/>\n")
	}
	b.WriteString("\t<key>StandardOutPath</key>\n\t<string>" + xmlText(spec.Log) + "</string>\n")
	b.WriteString("\t<key>StandardErrorPath</key>\n\t<string>" + xmlText(spec.Log) + "</string>\n")
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func xmlText(v string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(v)
}

// outside: an XML plist names its executable as `Program` or the first
// `ProgramArguments`; a binary plist is not read, so its program is unknown.
func programOf(body []byte) string {
	if !bytes.HasPrefix(bytes.TrimSpace(body), []byte("<?xml")) {
		return ""
	}
	d := xml.NewDecoder(bytes.NewReader(body))
	key, inArgs := "", false
	for {
		tok, err := d.Token()
		if err != nil {
			return ""
		}
		t, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch t.Name.Local {
		case "array":
			inArgs = key == "ProgramArguments"
		case "key", "string":
			var text string
			if err := d.DecodeElement(&text, &t); err != nil {
				return ""
			}
			if t.Name.Local == "key" {
				key = text
			} else if key == "Program" || inArgs {
				return text
			}
		}
	}
}
