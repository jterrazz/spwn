package machine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheMacOSNoiseMatchesByShapeAndAStrayIsStillReported(t *testing.T) {
	noise, err := os.ReadFile("macos/os-noise.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	allowed := filepath.Join(dir, "allowed.txt")
	created := filepath.Join(dir, "created.txt")
	station := ".config/station/preferences.json\n.local/state/station/runs/*\nLibrary/LaunchAgents/com.jterrazz.station.engine.plist\n"
	if err = os.WriteFile(allowed, append(noise, station...), 0o600); err != nil {
		t.Fatal(err)
	}
	listing := []string{
		"Library/Sharing/AirDropHashDB/AirDropHashDB.airdrop_dbv4.DqlFfj",
		"Library/Keychains/login.keychain-db.sb-378c204d-tBcNt7",
		"Library/DataDeliveryServices/ddsassertiondata.plist.sb-98e955cd-34PKkB",
		"Library/Mail/V10/MailData/Envelope Index.sb-12ab34cd-XyZ123",
		"Library/Application Scripts/group.com.apple.coreservices.useractivityd",
		".local/state/station/runs/api.pid",
		"stray-e2e-file",
		"Library/stray.plist",
		"Library/Keychains/login.keychain-db.bak",
		".config/station/stray",
		".config/station/preferences.json.sb-1a2b3c4d-AbCdEf",
	}
	if err = os.WriteFile(created, []byte(strings.Join(listing, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("/usr/bin/awk", "-f", "footprint.awk", allowed, created).Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "stray-e2e-file\nLibrary/stray.plist\nLibrary/Keychains/login.keychain-db.bak\n.config/station/stray\n.config/station/preferences.json.sb-1a2b3c4d-AbCdEf\n"
	if string(out) != want {
		t.Fatalf("unmatched:\n%s\nwant:\n%s", out, want)
	}
}

func TestARemovedPathMatchesOnlyUnderWhatTheActionMayRemoveNeverItsParent(t *testing.T) {
	dir := t.TempDir()
	may := filepath.Join(dir, "may.txt")
	removed := filepath.Join(dir, "removed.txt")
	if err := os.WriteFile(may, []byte(".cache/uv\nLibrary/Caches/Homebrew\n.zcompdump*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	listing := ".cache/uv\n.cache/uv/archive-v0\nLibrary/Caches/Homebrew/downloads\n.zcompdump-e2e-5.9\n.cache\nLibrary\n.zshrc\n.ssh/authorized_keys\n"
	if err := os.WriteFile(removed, []byte(listing), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("/usr/bin/awk", "-v", "under=1", "-f", "footprint.awk", may, removed).Output()
	if err != nil {
		t.Fatal(err)
	}
	if want := ".cache\nLibrary\n.zshrc\n.ssh/authorized_keys\n"; string(out) != want {
		t.Fatalf("unmatched:\n%s\nwant:\n%s", out, want)
	}
}
