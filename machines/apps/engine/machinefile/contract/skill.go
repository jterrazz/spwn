package contract

import "context"

type Skill interface {
	// invariant: the machine's `spwn` skill is written whole to every agent
	// skill folder of this machine's user, each copy landing at once (the
	// one before or the one after, never a mix), and only where its bytes
	// differ; changed says whether any copy was written. Its caller holds
	// the machine's turn, so two writers never meet.
	Write(ctx context.Context, body []byte) (changed bool, err error)
	// value: the skill as the last Write left it in the first folder;
	// false when none was ever written.
	Read(ctx context.Context) ([]byte, bool, error)
}
