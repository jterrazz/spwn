package calls

import (
	"cmp"
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const linuxDev = `machinefile: station.machinefile/v1
name: linux-dev
machine: { kind: workspace, system: { os: ubuntu, version: '24.04' } }
ports: { open: 3000-3999, to: private }
accounts: { share: [{ account: forge }, { account: mail }] }
`

const apiServer = `machinefile: station.machinefile/v1
name: api-server
machine: { kind: server, system: { os: ubuntu, version: '24.04' } }
`

type files struct{}

func (files) Read(context.Context) (machinefile.Tree, error) {
	return machinefile.Tree{Layer: machinefile.User, Files: []machinefile.File{
		{Path: "machinefiles/linux-dev.machine.yaml", Body: []byte(linuxDev)},
		{Path: "machinefiles/api-server.machine.yaml", Body: []byte(apiServer)},
	}}, nil
}

// value: every machine's calls as a fleet of a Mac, the mini, holding
// linux-1: the plan judged on linux-1, the ports read with no target, and
// the accounts the mini holds, asked of the mini; each call names where it
// was asked. ports fails the ports read beside its listing, bare leaves
// that listing empty, file names the Machinefile linux-1 follows, and a
// silent mini answers nothing, as the router reaches a VM through its Mac;
// unanswered, the mini leaves This Mac's ask of its ports unanswered, and
// answers the rest; macs rewrites the listing's Macs as they come back.
type fleet struct {
	asked      map[string]machine.Ref
	ports      error
	bare       bool
	file       string
	silent     bool
	unanswered bool
	macs       func([]ports.MacPorts) []ports.MacPorts
	macOf      string
}

var errSilent = errors.New("mini does not answer")

func (f *fleet) Invoke(_ context.Context, target machine.Ref, name string, _, out any, _ call.Progress) error {
	f.asked[name] = target
	switch out := out.(type) {
	case *machinefile.Plan:
		if target.Name != "linux-1" {
			return errors.New("the plan was judged off linux-1")
		}
		if f.silent {
			return errSilent
		}
		*out = machinefile.Plan{Name: cmp.Or(f.file, "linux-dev")}
	case *ports.Listing:
		if f.bare || f.silent {
			return f.ports
		}
		if f.unanswered {
			*out = ports.Listing{AllMachines: ports.AllMachines{Macs: []ports.MacPorts{
				{Name: "macbook-pro", ThisMac: true, Reachable: true, Shared: []ports.SharedPort{}},
				{Name: "mini", Shared: []ports.SharedPort{}},
			}}}
			return nil
		}
		on := func(port int) ports.Port {
			return ports.Port{Machine: machine.Ref{Name: "linux-1", On: "mini"}, Port: port}
		}
		listing := ports.Listing{AllMachines: ports.AllMachines{Macs: []ports.MacPorts{{Name: "mini", Reachable: true, Engine: ports.EngineInfo{Running: true}, Shared: []ports.SharedPort{
			{VM: "linux-1", GuestPort: 3000, To: catalog.AudiencePublic},
			{VM: "linux-1", GuestPort: 3001, To: catalog.AudiencePrivate},
			{VM: "linux-2", GuestPort: 3002, To: catalog.AudiencePublic},
		}, Ports: []ports.Port{on(3000), on(3500), on(8080), {Machine: machine.Ref{Name: "linux-2", On: "mini"}, Port: 3600}}}}}}
		if f.macs != nil {
			listing.AllMachines.Macs = f.macs(listing.AllMachines.Macs)
		}
		*out = listing
		return f.ports
	}
	return nil
}

func (f *fleet) Watch(_ context.Context, target machine.Ref, name string, _ any, _ call.Progress) (<-chan call.Update[any], error) {
	f.asked[name] = target
	if f.silent {
		return nil, errSilent
	}
	report := accounts.Report{}
	if target.Name == "mini" {
		report.Shares = []accounts.Shared{{Account: "forge", VM: "linux-1"}, {Account: "notes", VM: "linux-1"}, {Account: "mail", VM: "linux-2"}}
	}
	out := make(chan call.Update[any], 1)
	out <- call.Update[any]{Value: report}
	close(out)
	return out, nil
}

func fleetDeps(f *fleet, kind machine.Kind) Deps {
	return Deps{
		Name: "macbook-pro", Kind: kind, Far: f,
		Sources: func() []contract.Source { return []contract.Source{files{}} },
		MacOf: func(_ context.Context, name string) (string, bool) {
			return cmp.Or(f.macOf, "mini"), name == "linux-1"
		},
	}
}

func chapterOf(t *testing.T, d machinefile.Drift, c machinefile.Chapter) machinefile.ChapterDrift {
	t.Helper()
	for _, got := range d.Chapters {
		if got.Chapter == c {
			return got
		}
	}
	t.Fatalf("no chapter %s in %+v", c, d.Chapters)
	return machinefile.ChapterDrift{}
}

func TestAVMsDriftJudgesItsPackagesOnItAndItsSharesOnItsMac(t *testing.T) {
	// invariant: the machine is the input's: its plan is judged on it, its
	// ports read with no target and kept for it, its accounts asked of the
	// Mac it runs on, never of This Mac.
	f := &fleet{asked: map[string]machine.Ref{}}
	d, err := Drift.Run(context.Background(), fleetDeps(f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.asked["machinefile.plan"] != (machine.Ref{Name: "linux-1"}) || f.asked["ports.list"] != (machine.Ref{}) || f.asked["accounts.list"] != (machine.Ref{Name: "mini"}) {
		t.Errorf("asked %v; want the plan of linux-1, the ports with no target, the accounts of the mini", f.asked)
	}
	if d.Machine != "linux-1" {
		t.Errorf("machine %q", d.Machine)
	}
	p := chapterOf(t, d, machinefile.ChapterPorts)
	if p.Listed != 3 || p.Matching != 1 || len(p.Differing) != 2 || p.Differing[0].Port != 3000 || p.Differing[0].Seen != "public" || p.Differing[0].Asked != "private" {
		t.Errorf("ports %+v; want 3000 shared public where the file opens it to private", p)
	}
	if absent := p.Differing[1]; absent.Port != 3500 || absent.Condition != machinefile.Absent || absent.Seen != "" || absent.Asked != "private" {
		t.Errorf("ports differ as %+v; want 3500, listened inside the file's ports and not shared, absent", p.Differing)
	}
	a := chapterOf(t, d, machinefile.ChapterAccounts)
	if a.Listed != 2 || a.Matching != 1 || len(a.Differing) != 2 {
		t.Fatalf("accounts %+v; want forge matching, mail not copied, notes copied unasked", a)
	}
	if a.Differing[0].Account != "mail" || a.Differing[0].Condition != machinefile.Absent || a.Differing[1].Account != "notes" || a.Differing[1].Condition != machinefile.Extra {
		t.Errorf("accounts differ as %+v", a.Differing)
	}
	if d.Differences != 4 || d.Verdict.Sentence != "4 differences from linux-dev." {
		t.Errorf("drift %d, %q", d.Differences, d.Verdict.Sentence)
	}
}

func TestAVMWhoseSharesDoNotReadDriftsByItsPackagesAlone(t *testing.T) {
	// invariant: a VM whose Mac answers its plan but not its ports reads
	// its packages alone, and its verdict says where the rest is read.
	f := &fleet{asked: map[string]machine.Ref{}, ports: errors.New("no listing"), bare: true}
	d, err := Drift.Run(context.Background(), fleetDeps(f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Chapters) != 2 || d.Verdict.Sentence != "Matches linux-dev. Its ports and accounts are read from its Mac." {
		t.Errorf("%d chapters, %q", len(d.Chapters), d.Verdict.Sentence)
	}
}

func TestAVMWhoseMacLeftItsPortsUnansweredReadsThemUnread(t *testing.T) {
	// invariant: a ports listing whose VM's Mac did not answer This Mac
	// holds no port of that VM: it is unread, never an empty chapter.
	f := &fleet{asked: map[string]machine.Ref{}, unanswered: true}
	d, err := Drift.Run(context.Background(), fleetDeps(f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Chapters) != 2 || d.Verdict.Sentence != "Matches linux-dev. Its ports and accounts are read from its Mac." {
		t.Errorf("%d chapters %+v, %q; want its packages alone, its ports and accounts unread", len(d.Chapters), d.Chapters, d.Verdict.Sentence)
	}
}

func TestAVMsPortsReadOnlyWhereItsMacAnswersThem(t *testing.T) {
	// invariant: the ports chapter is judged only from a listing that
	// holds the VM's Mac answering, its engine running; any other listing
	// leaves it unread, never an empty chapter that reads as nothing shared.
	thisMac := ports.MacPorts{Name: "macbook-pro", ThisMac: true, Reachable: true, Engine: ports.EngineInfo{Running: true}, Shared: []ports.SharedPort{}}
	for _, c := range []struct {
		how    string
		f      *fleet
		judged bool
	}{
		{how: "the mini absent from a listing that came with an error", f: &fleet{ports: errors.New("tailnet status: not running"), macs: func([]ports.MacPorts) []ports.MacPorts {
			return []ports.MacPorts{thisMac}
		}}},
		{how: "the mini reachable, its engine down", f: &fleet{macs: func(m []ports.MacPorts) []ports.MacPorts {
			m[0].Engine.Running, m[0].Shared, m[0].Ports = false, []ports.SharedPort{}, nil
			return m
		}}},
		{how: "the mini named in another case", judged: true, f: &fleet{macs: func(m []ports.MacPorts) []ports.MacPorts {
			m[0].Name = "Mini"
			return m
		}}},
		{how: "a VM of This Mac, whose entry carries no name", judged: true, f: &fleet{macOf: "macbook-pro", macs: func(m []ports.MacPorts) []ports.MacPorts {
			m[0].Name, m[0].ThisMac = "", true
			return m
		}}},
	} {
		c.f.asked = map[string]machine.Ref{}
		d, err := Drift.Run(context.Background(), fleetDeps(c.f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.how, err)
		}
		if judged := len(d.Chapters) == 4; judged != c.judged {
			t.Errorf("%s: %d chapters %+v; want ports judged %v", c.how, len(d.Chapters), d.Chapters, c.judged)
		}
	}
}

func TestAPortsListingThatCameWithAnErrorIsStillJudged(t *testing.T) {
	// invariant: ports.list hands back its listing beside a failed tailnet
	// read; the listing's Macs are what the ports chapter judges.
	f := &fleet{asked: map[string]machine.Ref{}, ports: errors.New("tailnet status: not running")}
	d, err := Drift.Run(context.Background(), fleetDeps(f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p := chapterOf(t, d, machinefile.ChapterPorts); p.Listed != 3 || len(p.Differing) != 2 {
		t.Errorf("ports %+v; want the listing judged despite the tailnet's error", p)
	}
	if d.Differences != 4 {
		t.Errorf("drift %d, %q", d.Differences, d.Verdict.Sentence)
	}
}

func TestAServersPortsAndAccountsAreNotCounted(t *testing.T) {
	// invariant: a server's ports and accounts are what it runs, never its
	// file's: its drift has no such chapters and its verdict does not say
	// they are read elsewhere.
	f := &fleet{asked: map[string]machine.Ref{}, file: "api-server"}
	d, err := Drift.Run(context.Background(), fleetDeps(f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Chapters) != 2 || d.Differences != 0 || d.Verdict.Sentence != "Matches api-server." {
		t.Errorf("%d chapters, %d differences, %q; want the packages alone, matching", len(d.Chapters), d.Differences, d.Verdict.Sentence)
	}
	if _, asked := f.asked["ports.list"]; asked {
		t.Error("a server's ports were read")
	}
}

func TestAVMWhoseMacIsSilentIsNotRead(t *testing.T) {
	// invariant: a VM is reached through its Mac: one whose Mac does not
	// answer has no packages to judge, and its drift is the Mac's silence.
	f := &fleet{asked: map[string]machine.Ref{}, silent: true}
	_, err := Drift.Run(context.Background(), fleetDeps(f, machine.Computer), DriftIn{Machine: "linux-1"}, nil)
	if !errors.Is(err, errSilent) {
		t.Errorf("drift of a VM of a silent Mac = %v; want its Mac's silence", err)
	}
}
