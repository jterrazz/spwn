package catalog

import "time"

type CheckedSetting struct {
	// value: a setting station reads from macOS and never applies: what the
	// machine should have, the facts it is judged on, and how the owner
	// fixes it.
	Name, Label, Description string
	Kinds                    []MachineKind
	// value: the facts it is judged on, the first its own.
	Facts []Fact
	// value: where the owner fixes it, a System Settings path or a step
	// outside station; never a station command.
	Fix string
	// value: the System Settings pane Fix names, as the URL that opens it;
	// empty for a fix outside System Settings.
	Pane string
	// value: how long its facts may take; past it the row is Unknown.
	Budget time.Duration
	// value: the line that turns it on from Terminal, the owner's to type
	// (station never changes a security setting); empty when only Fix does.
	Command string
	// value: what the owner does, as the hint that shows Command says it.
	Turn string
}

const systemSettings = "x-apple.systempreferences:"

const (
	checkedBudget = 2 * time.Second
	updatesBudget = 10 * time.Second
)

// value: the kind a checked setting lives on: a Mac itself.
var macKind = []MachineKind{Mac}

var CheckedSettings = []CheckedSetting{
	{Name: "filevault", Label: "Disk encryption", Description: "FileVault encrypts the disk", Kinds: macKind, Facts: []Fact{FactFileVault}, Fix: "System Settings > Privacy & Security > FileVault", Pane: systemSettings + "com.apple.settings.PrivacySecurity.extension?FileVault", Budget: checkedBudget},
	{Name: "firewall", Label: "Firewall", Description: "The macOS firewall refuses unasked connections", Kinds: macKind, Facts: []Fact{FactFirewall}, Fix: "System Settings > Network > Firewall", Pane: systemSettings + "com.apple.Network-Settings.extension?Firewall", Budget: checkedBudget,
		Command: AsRoot("/usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on"), Turn: "Turn the firewall on"},
	{Name: "sip", Label: "System Integrity Protection", Description: "SIP keeps the system files read-only", Kinds: macKind, Facts: []Fact{FactSIP}, Fix: "Recovery > Terminal > csrutil enable", Budget: checkedBudget},
	{Name: "automatic-updates", Label: "Automatic security updates", Description: "macOS installs its security responses by itself", Kinds: macKind, Facts: []Fact{FactAutoUpdates}, Fix: "System Settings > General > Software Update > Automatic updates", Pane: systemSettings + "com.apple.Software-Update-Settings.extension", Budget: checkedBudget},
	{Name: "screen-lock", Label: "Screen lock", Description: "The screen locks after 5 min idle (a desktop: 5 min after the display turns off)", Kinds: macKind, Facts: []Fact{FactScreenLock, FactDisplaySleep}, Fix: "System Settings > Lock Screen", Pane: systemSettings + "com.apple.Lock-Screen-Settings.extension", Budget: checkedBudget},
	{Name: "gatekeeper", Label: "Gatekeeper", Description: "Gatekeeper opens only signed apps", Kinds: macKind, Facts: []Fact{FactGatekeeper}, Fix: "System Settings > Privacy & Security > Security", Pane: systemSettings + "com.apple.settings.PrivacySecurity.extension?Security", Budget: checkedBudget},
	{Name: "remote-login", Label: "Remote login", Description: "Remote login is off on a laptop, on on an always-on Mac", Kinds: macKind, Facts: []Fact{FactRemoteLogin}, Fix: "System Settings > General > Sharing > Remote Login", Pane: systemSettings + "com.apple.Sharing-Settings.extension", Budget: checkedBudget},
	{Name: "time-machine", Label: "Time Machine", Description: "Time Machine's last backup is within 7 days", Kinds: macKind, Facts: []Fact{FactTimeMachine}, Fix: "System Settings > General > Time Machine", Pane: systemSettings + "com.apple.Time-Machine-Settings.extension", Budget: checkedBudget},
	{Name: "macos-updates", Label: "macOS updates", Description: "No macOS update is waiting", Kinds: macKind, Facts: []Fact{FactSystemUpdates}, Fix: "System Settings > General > Software Update", Pane: systemSettings + "com.apple.Software-Update-Settings.extension", Budget: updatesBudget},
}
