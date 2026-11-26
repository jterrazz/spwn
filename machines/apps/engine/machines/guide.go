package machines

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/guide"
)

const (
	// value: the act family of a create: a VM made, its image built first
	// when it is missing, its id `machines.create:<name>`.
	GuideCreate = "machines.create"

	// value: how a create names itself in its Guide's heading.
	guideCreateAct = "New machine"
)

func (s BirthStep) Label() string {
	switch s {
	case BornBuilding:
		return "Image"
	case BornCloning:
		return "Disk"
	case BornBooting:
		return "Boot"
	case BornAddress:
		return "Network"
	case BornSSH:
		return "SSH"
	}
	return string(s)
}

func CreateRail(build []guide.Rung) []guide.Rung {
	// value: a create's steps: its image's build's own, when it builds one,
	// then its disk, its first run, its address and its ssh.
	out := slices.Clone(build)
	for _, s := range []BirthStep{BornCloning, BornBooting, BornAddress, BornSSH} {
		out = append(out, guide.Rung{ID: string(s), Label: s.Label()})
	}
	return out
}

func CreateGuide(a guide.Act, vm *VM) guide.Guide {
	// invariant: a create's Guide says only what its act and its VM's
	// record hold: a step reached, a line its call reported, what the
	// failure kept, and the one repair the VM's own gestures offer. Birth's
	// per mille is weights, not a measure: the Guide draws a live line, never
	// a bar.
	if failed := vm != nil && vm.Birth != nil && vm.Birth.Failed; failed && a.Ended == guide.EndedFailed {
		a.Step = string(vm.Birth.Step)
	}
	at := slices.IndexFunc(a.Rail, func(r guide.Rung) bool { return r.ID == a.Step })
	label, next := a.Step, ""
	if at >= 0 {
		label = a.Rail[at].Label
	}
	if at >= 0 && at+1 < len(a.Rail) {
		next = a.Rail[at+1].Label
	}
	g := guide.Guide{ID: a.ID, Act: guideCreateAct, Subject: a.Subject, Steps: guide.Rail(a), Ended: a.Ended}
	switch a.Ended {
	case "":
		g.Live = true
		g.Line = "Creating " + a.Subject + ": " + label
		g.View = guide.View{
			Illustration: pictureOf(a, at), Headline: guide.Headline{Now: label + " now", Next: nextWords(next)},
			Sentence: "Spwn is creating " + a.Subject + ". It goes on if you hide this.", Move: guide.MoveEngine,
			Progress: &guide.Progress{Kind: guide.Live, Line: orLabel(a.Line, label)},
		}
	case guide.EndedDone:
		g.Line = a.Subject + " is ready"
		g.View = guide.View{
			Illustration: guide.PictureDone, Headline: guide.Headline{Now: a.Subject + " is ready", Next: ""},
			Sentence: readyWords(a.Subject, vm), Move: guide.MoveOwner,
			Action: &guide.Gesture{Label: "Open " + a.Subject, Nav: "machine:" + a.Subject},
		}
	default:
		g.View = endedView(a, vm, at, label)
		g.Line = "Creating " + a.Subject + " " + map[guide.Ended]string{guide.EndedFailed: "failed", guide.EndedInterrupted: "was interrupted"}[a.Ended] + " at " + label
		if a.Ended == guide.EndedInterrupted && at >= 0 {
			g.Steps[at].Detail = guide.EndedInterrupted.Word()
		}
	}
	return g
}

func endedView(a guide.Act, vm *VM, at int, label string) guide.View {
	picture := guide.PictureBootFailed
	if building(a, at) {
		picture = guide.PictureBuildFailed
	}
	v := guide.View{Illustration: picture, Headline: guide.Headline{Now: label + " failed", Next: ""}, Sentence: a.Cause, Move: guide.MoveEngine}
	if a.Ended == guide.EndedInterrupted {
		v.Headline.Now = label + " was interrupted"
		v.Sentence = "The engine stopped while it created " + a.Subject + "."
	}
	switch {
	case vm == nil:
		v.Kept = "Nothing was made."
	case vm.Birth != nil && vm.Birth.Failed && vm.Birth.Gone:
		v.Kept, v.Move = "Its disk is kept until you delete it.", guide.MoveOwner
		v.Action = deleteGesture(vm.Name, vm.Mac)
	case vm.State == Running:
		v.Kept, v.Move = vm.Name+" runs, but its set-up did not finish.", guide.MoveOwner
		v.Action = deleteGesture(vm.Name, vm.Mac)
		if vm.Birth != nil && vm.Birth.Failed {
			v.Action = &guide.Gesture{Label: "Try again", Call: "machines.start", Args: map[string]string{"VM": vm.Name}, On: vm.Mac}
			v.Quiet = deleteGesture(vm.Name, vm.Mac)
		}
	default:
		v.Kept, v.Move = keptWords(vm), guide.MoveOwner
		if gestures := GesturesOf(*vm); len(gestures) > 0 && (gestures[0] == GestureTryAgain || gestures[0] == GestureStart) {
			v.Action = &guide.Gesture{Label: map[Gesture]string{GestureTryAgain: "Try again", GestureStart: "Start " + vm.Name}[gestures[0]], Call: "machines.start", Args: map[string]string{"VM": vm.Name}, On: vm.Mac}
		}
		v.Quiet = deleteGesture(vm.Name, vm.Mac)
	}
	if v.Sentence == "" && vm != nil && vm.Birth != nil {
		v.Sentence = vm.Birth.Cause
	}
	return v
}

func keptWords(vm *VM) string {
	if vm.Detail == HalfMade {
		return vm.Name + " is kept, stopped, half made."
	}
	return vm.Name + " is kept, stopped."
}

func deleteGesture(name, mac string) *guide.Gesture {
	return &guide.Gesture{Label: "Delete " + name, Call: "machines.delete", Args: map[string]string{"VM": name}, On: mac, Confirm: name}
}

func readyWords(name string, vm *VM) string {
	if vm == nil || vm.Mac == "" {
		return name + " runs."
	}
	return name + " runs on " + vm.Mac + "."
}

func nextWords(next string) string {
	if next == "" {
		return "Ready next"
	}
	return next + " next"
}

func orLabel(line, label string) string {
	if line == "" {
		return label
	}
	return line
}

func building(a guide.Act, at int) bool {
	return at >= 0 && !slices.Contains(BornBuilding.Enum(), a.Rail[at].ID)
}

func pictureOf(a guide.Act, at int) guide.Picture {
	// value: a build's steps draw its four stages in turn, every later step
	// the VM's start.
	if stages := []guide.Picture{guide.PictureBuild1, guide.PictureBuild2, guide.PictureBuild3, guide.PictureBuild4}; building(a, at) {
		return stages[min(at, len(stages)-1)]
	}
	return guide.PictureBoot
}
