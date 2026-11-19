package background

const (
	ScopeUser = "user"
	ScopeRoot = "root"
)

type Job struct {
	Label string
	// value: the program the definition runs, as the definition names it.
	Program string
	// value: the job's definition on disk; empty for a job loaded without one.
	File  string
	Scope string
	PID   int
	// value: the last exit code, kept only while the job is not running.
	LastExit int
	Running  bool
	Loaded   bool
	// value: the job manager did not answer for this job: its state is unknown.
	Unread        bool
	ProgramExists bool
	OwnerExists   bool
	// value: the command line the owner types to remove the job by hand.
	Removal string
}
