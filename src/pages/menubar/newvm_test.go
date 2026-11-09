package menubar

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
)

func field(fields []page.Field, key string) page.Field {
	for _, f := range fields {
		if f.Key == key {
			return f
		}
		if inner := field(f.Fields, key); inner.Key != "" {
			return inner
		}
	}
	return page.Field{}
}

// invariant: SPEC A5 New VM: from home's + it opens on the roomiest Mac
// and says so, every Mac's own form beside it, the VM's next free name
// refused once taken.
func TestNewVMOpensOnTheRoomiestMacWithTheNextFreeName(t *testing.T) {
	w := newHeroWorld()
	w.mini().Hardware = hardware(96)
	p := homeOf(t, w)
	a := actionOf(p, ActionNewVM)
	if a.Sheet != page.SheetNewVM || a.Key != "cmd+n" || a.Run.Target == nil || a.Run.Target.Name != heroMini {
		t.Fatalf("new-vm %+v, want the sheet on ⌘N, run on %s", a, heroMini)
	}
	if a.Form.Subtitle != "On mac-mini, the Mac with the most free memory. Ready in about a minute." {
		t.Errorf("subtitle %q", a.Form.Subtitle)
	}
	name := field(a.Form.Fields, "Request.Name")
	if name.Default != "linux-2" || !slices.Contains(name.Taken, "linux-1") || name.TakenHud != "{Request.Name} already exists" {
		t.Errorf("name %+v, want linux-2, linux-1 refused", name)
	}
	on := field(a.Form.Fields, keyOn)
	if !on.Swaps || len(on.Options) != 2 || on.Options[1].Value != ActionNewVM || actionOf(p, on.Options[0].Value).Sheet != page.SheetNewVM {
		t.Errorf("On %+v, want each Mac's own form", on.Options)
	}
}

// invariant: SPEC A9 1-8 and 1-9: the memory past what the Mac has free
// is refused with its words; the accounts it can share are checked,
// Share for showing only while one is.
func TestNewVMOffersTheMemoryAndTheAccountsOfItsMac(t *testing.T) {
	w := newHeroWorld()
	w.r.Accounts.Macs = []accounts.MacAccounts{{Name: heroPro, ThisMac: true, Accounts: []accounts.Account{
		{Provider: catalog.ProviderGitHub, State: catalog.StateOK},
		{Provider: catalog.ProviderClaude, State: catalog.StateOK},
	}}}
	a := actionOf(homeOf(t, w), ActionNewVMOn+heroPro)
	memory := field(a.Form.Fields, "Request.Size.MemoryBytes")
	free := memory.Limit
	if free <= 0 || memory.LimitNote == "" || slices.ContainsFunc(memory.Options, func(o page.Option) bool { return o.Disabled != "" }) != (free < 32) {
		t.Errorf("memory %+v", memory)
	}
	share := field(a.Form.Fields, keyShare)
	if share.Default != "github,claude" || len(share.Options) != 2 {
		t.Errorf("share %+v, want GitHub and Claude checked", share)
	}
	if f := field(a.Form.Fields, keyShareFor); f.When == nil || f.When.Key != keyShare || !f.When.Set {
		t.Errorf("Share for %+v, want shown only while an account is checked", f)
	}
}

// invariant: a server's form names it first and says that a laptop's
// sleep pauses it; its Create sends machines.create-server.
func TestANewServerSaysItsMacsSleepPausesIt(t *testing.T) {
	w := newHeroWorld()
	w.r.All.Macs[0].Laptop = true
	p := homeOf(t, w)
	a := actionOf(p, ActionNewServerOn+heroPro)
	if a.Run.Call != CallCreateServer || a.Form.Fields[1].Key != "Name" || a.Form.Warn != "This Mac may sleep: a server pauses with it." {
		t.Errorf("server %+v", a)
	}
	if actionOf(p, ActionNewServerOn+heroMini).Form.Warn != "" {
		t.Errorf("a desktop's server pauses with nothing")
	}
	open := actionOf(p, ActionCreateOpen+a.ID)
	if open.Run.Then == nil || !slices.Equal(open.Run.Then.Argv, []string{"--on", "{Name}"}) {
		t.Errorf("split %+v, want the terminal on what it made", open.Run)
	}
}
