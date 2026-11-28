package fake

import (
	"context"
	"io/fs"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: the `spwn` skill of a machine of the world, kept in its stores,
// so a story reads what an apply wrote and how often its bytes changed.

type Skill struct {
	w       *world.World
	machine string
}

var _ contract.Skill = Skill{}

func SkillOf(w *world.World, machine string) Skill { return Skill{w: w, machine: machine} }

func (s Skill) Write(_ context.Context, body []byte) (bool, error) {
	// invariant: a scripted Skill.Write fault is a skills folder the user
	// made read-only: the write is refused as the file system refuses it,
	// and the skill stays as it was.
	s.w.Lock()
	defer s.w.Unlock()
	if s.w.Fails("Skill.Write") {
		return false, &fs.PathError{Op: "open", Path: catalog.SpwnSkillFiles[0], Err: fs.ErrPermission}
	}
	st := s.w.Mac(s.machine).Store()
	if st.SpwnSkill == string(body) {
		return false, nil
	}
	st.SpwnSkill = string(body)
	st.SpwnSkillWrites++
	return true, nil
}

func (s Skill) Read(context.Context) (body []byte, ok bool, err error) {
	s.w.Lock()
	defer s.w.Unlock()
	text := s.w.Mac(s.machine).Store().SpwnSkill
	return []byte(text), text != "", nil
}
