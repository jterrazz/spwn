package main

import (
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const bundleAgentsDir = "apps/desktop/src-tauri/agents"

// value: the definitions of the engine's two jobs inside Spwn.app (ADR-073,
// mode A): static, since a sealed bundle is never rewritten, so they hold no
// home, no environment and no log path; the engine sets those itself at its
// start. SMAppService registers them, and launchd runs BundleProgram, a path
// relative to the bundle.
func bundledAgents() map[string][]byte {
	return map[string][]byte{
		path.Base(catalog.BundledPlist(catalog.EngineLabel)): bundledAgent(catalog.EngineLabel, catalog.EngineArgs),
		path.Base(catalog.BundledPlist(catalog.AgentLabel)):  bundledAgent(catalog.AgentLabel, catalog.AgentArgs),
	}
}

func bundledAgent(label string, args []string) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	b.WriteString("\t<key>Label</key>\n\t<string>" + label + "</string>\n")
	b.WriteString("\t<key>BundleProgram</key>\n\t<string>" + catalog.BundledEnginePath + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, arg := range append([]string{catalog.BundledEngine}, args...) {
		b.WriteString("\t\t<string>" + arg + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n\t<key>KeepAlive</key>\n\t<true/>\n\t<key>AbandonProcessGroup</key>\n\t<true/>\n")
	b.WriteString("</dict>\n</plist>\n")
	return []byte(b.String())
}
