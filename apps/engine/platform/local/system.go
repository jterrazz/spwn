package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"time"
)

var (
	ErrNotFound    = errors.New("command not found")
	ErrTimeout     = errors.New("command timed out")
	ErrSilent      = errors.New("command said nothing")
	ErrCrossDevice = errors.New("another volume: a rename cannot move it")
	ErrWildcard    = errors.New("refused: station never listens on every interface")
)

type Stream int

const (
	ToStdout Stream = iota
	ToStderr
)

type Cmd struct {
	Name    string
	Args    []string
	Dir     string
	Env     []string
	Stdin   string
	Timeout time.Duration
	// value: how long an attached command may say nothing before its first
	// word; past it the command is stopped and its run fails as ErrSilent.
	Answer time.Duration
	Track  string
	// value: the PATH Env names is the child's whole PATH: no landing
	// folder is added to it.
	Sealed bool
	// value: the files an attached command's stdout and stderr are appended
	// to, one per item it serves, in place of the terminal; its stdin stays.
	Logs []string
}

type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Attached struct {
	ExitCode   int
	LastStderr string
}

type Sig int

const (
	SigTerm Sig = iota
	SigKill
)

type Runner interface {
	Output(ctx context.Context, c Cmd) (Output, error)
	Attach(ctx context.Context, c Cmd, out Stream) (Attached, error)
	Start(c Cmd, logPath string) (pid int, err error)
	LookPath(name string) (string, bool)
	// value: a Runner whose children are given env alone, never the
	// environment station was started with, and whose lookup walks env's
	// PATH where this one walks station's.
	Within(env []string) Runner
	Alive(pid int) bool
	Signal(pid int, sig Sig) error
	Reap(dir string) int
}

type FS interface {
	fs.FS
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// value: the file at path replaced whole, at once and durably, with its
	// own mode and owner; one not there yet is made with perm.
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	Rename(from, to string) error
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Glob(pattern string) ([]string, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Chmod(path string, perm fs.FileMode) error
	Chown(path string, uid, gid int) error
	Readlink(path string) (string, error)
}

type Env interface {
	Executable() string
	Home() string
	User() string
	Shell() string
	Hostname() string
	Euid() int
	GOOS() string
	GOARCH() string
}

type Net interface {
	Dial(ctx context.Context, mac string, port int, timeout time.Duration) bool
	Get(ctx context.Context, url string, header map[string]string) (status int, body []byte, err error)
	// value: a GET's answer with the headers it came with, for a caller
	// that reads them (a rate limit's end).
	Fetch(ctx context.Context, url string, header map[string]string) (Reply, error)
	// value: one UDP datagram broadcast to 255.255.255.255, port: macOS sends
	// it out of the interface of the default route only.
	Broadcast(ctx context.Context, port int, payload []byte) error
	// value: the hardware address of the interface that holds ip, as
	// aa:bb:cc:dd:ee:ff; empty when none does.
	HardwareAddress(ip string) string
}

type Reply struct {
	Status int
	// value: each header's first value, by its canonical name
	// (`X-Ratelimit-Remaining`); Said reads one by its name in any case.
	Header map[string]string
	Body   []byte
}

func (r Reply) Said(name string) string { return r.Header[http.CanonicalHeaderKey(name)] }

type Conn interface{ io.ReadWriteCloser }

type Listener interface {
	Accept() (Conn, error)
	Close() error
	Port() int
}

type Sockets interface {
	ListenUnix(path string) (Listener, error)
	DialUnix(ctx context.Context, path string, timeout time.Duration) (Conn, error)
	// value: a TCP connection to address (host:port), for a protocol a
	// library speaks over it.
	DialTCP(ctx context.Context, address string, timeout time.Duration) (net.Conn, error)
	ListenTCP(ip string, port int) (Listener, error)
	Pipe(ctx context.Context, ip string, port int, in io.Reader, out io.Writer) error
	Join(a, b Conn)
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type System struct {
	Run     Runner
	FS      FS
	Env     Env
	Net     Net
	Sockets Sockets
	Clock   Clock
}
