package catalog

const systemSettings = "x-apple.systempreferences:"

type Pane struct {
	// value: a System Settings pane as the URL that opens it, and what the
	// button that opens it says it opens.
	URL, Object string
}

const (
	// value: each pane id was read on disk in macOS 26.6.2's
	// System Settings extension of that id (allowsXAppleSystemPreferencesURLScheme);
	// an anchor is kept only when the extension's own search terms declare it,
	// so Local Network opens its pane's top.
	paneFileVault    = systemSettings + "com.apple.settings.PrivacySecurity.extension?FileVault"
	paneFirewall     = systemSettings + "com.apple.Network-Settings.extension?Firewall"
	paneSecurity     = systemSettings + "com.apple.settings.PrivacySecurity.extension?Security"
	PaneLocalNetwork = systemSettings + "com.apple.settings.PrivacySecurity.extension"
	paneSoftware     = systemSettings + "com.apple.Software-Update-Settings.extension"
	paneLockScreen   = systemSettings + "com.apple.Lock-Screen-Settings.extension"
	paneSharing      = systemSettings + "com.apple.Sharing-Settings.extension"
	paneTimeMachine  = systemSettings + "com.apple.Time-Machine-Settings.extension"
)

var Panes = []Pane{ // value: every pane the window may open, the one list the shell's opener admits.
	{paneFileVault, "FileVault settings"},
	{paneFirewall, "Firewall settings"},
	{paneSecurity, "Security settings"},
	{PaneLocalNetwork, "Local Network settings"},
	{paneSoftware, "Software Update settings"},
	{paneLockScreen, "Lock Screen settings"},
	{paneSharing, "Sharing settings"},
	{paneTimeMachine, "Time Machine settings"},
}

func PaneObject(url string) string {
	for _, p := range Panes {
		if p.URL == url {
			return p.Object
		}
	}
	return ""
}
