package catalog_test

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheBuiltInImagesAreTheThreeBases(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, kind := range map[string]catalog.MachineKind{"base-ubuntu": catalog.Linux, "base-debian": catalog.Linux, "base-macos": catalog.MacOS} {
		if img, ok := c.Image(name); !ok || img.Machinefile != name || img.Kind != kind {
			t.Errorf("%s: %+v %v, want the built-in %s image made from its own Machinefile", name, img, ok, kind)
		}
	}
	for name, want := range map[string][2]string{"base-ubuntu": {"ubuntu", "24.04"}, "base-debian": {"debian", "13"}, "base-macos": {"macos", "26.6.2"}} {
		img, _ := c.Image(name)
		if got := [2]string{string(img.Distro), img.DistroVersion}; got != want {
			t.Errorf("%s is %v, want the distro %v, its version its Machinefile's", name, got, want)
		}
	}
	for _, old := range []string{"linux", "station-linux", "macos"} {
		if img, ok := c.Image(old); ok {
			t.Errorf("%s names %s, want no image", old, img.Name)
		}
	}
}

func TestATemplateIsBuiltWithThePlumbingOfItsSystem(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	for os, want := range map[catalog.OS]string{catalog.OSUbuntu: "base-ubuntu", "": "base-ubuntu", catalog.OSDebian: "base-debian", catalog.OSMacOS: ""} {
		img, ok := c.RecipeFor(os)
		if img.Name != want || ok != (want != "") {
			t.Errorf("the recipe for %q is %q %v; want %q", os, img.Name, ok, want)
		}
	}
}

func TestAFetchOnlyBaseIsPinnedByDigest(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	fetched := 0
	for _, img := range c.Images {
		if !c.FetchedOnly(img.Name) {
			continue
		}
		fetched++
		file := path.Join("machines", img.Machinefile+catalog.MachinefileSuffix)
		body, err := fs.ReadFile(catalog.BuiltinFiles(), file)
		if err != nil {
			t.Fatal(err)
		}
		mf, err := catalog.DecodeMachinefile(file, body)
		if err != nil {
			t.Fatal(err)
		}
		if from := mf.VM.Image.From; !catalog.PinnedByDigest(from) {
			t.Errorf("%s is fetched from %q, want its upstream pinned by digest", img.Name, from)
		}
		if img.DistroVersion == "" {
			t.Errorf("%s says no system.version, want the pinned image's version", img.Name)
		}
	}
	if fetched == 0 {
		t.Error("no built-in image is fetched only, want base-macos")
	}
}

func TestAFetchOnlyBaseCarriesTheFirstLoginThatPutsStationsKeyIn(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, img := range c.Images {
		if c.FetchedOnly(img.Name) != (img.FirstLogin != "") {
			t.Errorf("%s: fetched only %v, first login %q; want a first login exactly when it is fetched only", img.Name, c.FetchedOnly(img.Name), img.FirstLogin)
			continue
		}
		if img.FirstLogin == "" {
			continue
		}
		script, err := fs.ReadFile(catalog.Recipes, img.FirstLogin)
		if err != nil {
			t.Fatal(err)
		}
		lockdownPutsTheKeyInLastAndItsMarkerAfterIt(t, img.Name, string(script))
		lockdownClosesTheImageBeforeAnythingElse(t, img.Name, string(script))
	}
}

func lockdownClosesTheImageBeforeAnythingElse(t *testing.T, name, script string) {
	t.Helper()
	checked := strings.Index(script, `= "$BIN_SHA256" ]`)
	if checked < 0 {
		t.Fatalf("%s's first login never checks its binary", name)
	}
	firstOther := len(script)
	for _, after := range []string{"sysadminctl -addUser", "/usr/local/bin/station", "createhomedir", "/etc/sudoers.d/60-station-user", "ssh-keygen -A"} {
		at := strings.Index(script, after)
		if at < 0 {
			t.Fatalf("%s's first login lacks %q", name, after)
		}
		firstOther = min(firstOther, at)
	}
	for _, step := range []string{
		"PasswordAuthentication no",
		"KbdInteractiveAuthentication no",
		"PermitRootLogin no",
		"autoLoginUser",
		"rm -f /etc/kcpassword",
		"/bin/launchctl disable system/com.apple.screensharing",
		"/bin/launchctl bootout system/com.apple.screensharing",
		"/bin/rm -f /Library/Preferences/com.apple.VNCSettings.txt",
		`-passwd "/Users/$LOGIN_USER"`,
		`: > "$login_home/.ssh/authorized_keys"`,
		`rm -f "/etc/sudoers.d/$LOGIN_USER-nopasswd"`,
	} {
		at := strings.Index(script, step)
		switch {
		case at < 0:
			t.Errorf("%s's first login lacks %q", name, step)
		case at < checked:
			t.Errorf("%s's first login runs %q before it checks its binary", name, step)
		case at > firstOther:
			t.Errorf("%s's first login sets the guest up before %q: a failure between leaves the image's login open", name, step)
		}
	}
	lock := strings.Index(script, "/etc/ssh/sshd_config.d/000-station.conf")
	for _, step := range []string{
		"vm_user_rule='",
		`[[ "$VM_USER" =~ $vm_user_rule ]]`,
		`[ "$VM_USER" != "$LOGIN_USER" ]`,
		`[ "$VM_USER" != root ]`,
		`vm_uid=$(/usr/bin/id -u "$VM_USER" 2>/dev/null)`,
		`{ [ "$vm_uid" -lt 500 ] || [ "$vm_uid" -gt 60000 ]; }`,
		`[ "$vm_uid" = "$(/usr/bin/id -u "$LOGIN_USER" 2>/dev/null)" ]`,
	} {
		at := strings.Index(script, step)
		switch {
		case at < 0:
			t.Errorf("%s's first login lacks %q, the VM user's rule", name, step)
		case at < checked:
			t.Errorf("%s's first login runs %q before it checks its binary", name, step)
		case at > lock:
			t.Errorf("%s's first login checks its VM user by %q after it locks the image's login: a refusal there leaves a guest no login opens", name, step)
		}
	}
	for _, step := range []string{
		"/usr/sbin/chown root:wheel /usr/local/bin",
		"/bin/chmod 0755 /usr/local/bin",
		`/usr/sbin/visudo -cf "$rule"`,
		`/bin/mv -f "$rule" /etc/sudoers.d/60-station-user`,
	} {
		if !strings.Contains(script, step) {
			t.Errorf("%s's first login lacks %q", name, step)
		}
	}
	if strings.Contains(script, "> /etc/sudoers.d/60-station-user") {
		t.Errorf("%s's first login writes its sudo rule in place, read by sudo before visudo checked it", name)
	}
}

func lockdownPutsTheKeyInLastAndItsMarkerAfterIt(t *testing.T, name, script string) {
	t.Helper()
	key := strings.Index(script, `"$IMAGE_KEY" > "$home/.ssh/authorized_keys"`)
	marker := strings.Index(script, `: > "$marker"`)
	if key < 0 || marker < 0 {
		t.Fatalf("%s's first login writes no key (%d) or no marker (%d)", name, key, marker)
	}
	for _, step := range []string{
		"rm -f /etc/kcpassword",
		"/bin/launchctl disable system/com.apple.screensharing",
		"/bin/launchctl bootout system/com.apple.screensharing",
		"/bin/rm -f /Library/Preferences/com.apple.VNCSettings.txt",
		`-passwd "/Users/$LOGIN_USER"`,
		`-passwd "/Users/$VM_USER"`,
		`: > "$home/.ssh/authorized_keys"`,
		`: > "$login_home/.ssh/authorized_keys"`,
		`rm -f "/etc/sudoers.d/$LOGIN_USER-nopasswd"`,
		"PasswordAuthentication no",
		"KbdInteractiveAuthentication no",
		"PermitRootLogin no",
		"rm -f /etc/ssh/ssh_host_*",
		"/usr/bin/ssh-keygen -A",
		"/usr/sbin/sshd -T",
		`-authonly "$LOGIN_USER" "$LOGIN_PASSWORD"`,
	} {
		at := strings.LastIndex(script, step)
		switch {
		case at < 0:
			t.Errorf("%s's first login lacks %q: what the image left open would outlive it", name, step)
		case at > key:
			t.Errorf("%s's first login puts the key in before %q: a failure between leaves the guest open and keyed", name, step)
		}
	}
	if marker < key {
		t.Errorf("%s's first login writes its marker before the key", name)
	}
	for _, line := range strings.Split(strings.TrimSpace(script[marker:]), "\n")[1:] {
		if !strings.HasSuffix(strings.TrimSpace(line), ">&3") {
			t.Errorf("%s's first login runs %q after its marker, which is its last step", name, line)
		}
	}
}

func TestTheLinuxPlumbingRefusesASystemAccountBeforeItTouchesIt(t *testing.T) {
	body, err := fs.ReadFile(catalog.Recipes, "images/linux.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(body)
	check := strings.Index(script, `vm_uid=$(id -u "$VM_USER" 2>/dev/null)`)
	refusal := strings.Index(script, `{ [ "$vm_uid" -lt 1000 ] || [ "$vm_uid" -gt 60000 ]; }`)
	if check < 0 || refusal < 0 {
		t.Fatal("linux.sh never refuses a VM user that is an existing account outside 1000 to 60000, a system account (daemon, nobody)")
	}
	login := strings.Index(script, `[ "$vm_uid" = "$(id -u admin 2>/dev/null)" ]`)
	if login < 0 {
		t.Fatal("linux.sh never refuses a VM user that is the base's own admin account under another name, by its uid")
	}
	for _, step := range []string{"useradd", "/etc/sudoers.d/60-station-user", "authorized_keys"} {
		if at := strings.Index(script, step); at >= 0 && (at < refusal || at < login) {
			t.Errorf("linux.sh runs %q before it refuses a system account, or the base's admin, as its VM user", step)
		}
	}
}

func TestTheLinuxPlumbingRefusesAdminByNameWhateverTheBaseHolds(t *testing.T) {
	body, err := fs.ReadFile(catalog.Recipes, "images/linux.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(body)
	byName := strings.Index(script, `[ "$VM_USER" = admin ]`)
	if byName < 0 {
		t.Fatal("linux.sh never refuses the VM user admin by its name: on a base without that account it would make it, and linux-cleanup.sh would delete it")
	}
	if at := strings.Index(script, "useradd"); at >= 0 && at < byName {
		t.Error("linux.sh makes the VM user before it refuses admin by name")
	}
}

func TestADigestPinsAndATagDoesNot(t *testing.T) {
	digest := "sha256:65c3133a633fda8753cf1f71f2d55727792d534a2125b0032466563dcc8e58fd"
	for ref, want := range map[string]bool{
		"ghcr.io/cirruslabs/macos-tahoe-base@" + digest:                                                                    true,
		"ghcr.io/cirruslabs/macos-tahoe-base:latest":                                                                       false,
		"ghcr.io/cirruslabs/macos-tahoe-base@sha256:65c3":                                                                  false,
		"ghcr.io/cirruslabs/macos-tahoe-base@sha256:" + "65C3133A633FDA8753CF1F71F2D55727792D534A2125B0032466563DCC8E58FD": false,
	} {
		if got := catalog.PinnedByDigest(ref); got != want {
			t.Errorf("PinnedByDigest(%q) = %v, want %v", ref, got, want)
		}
	}
}
