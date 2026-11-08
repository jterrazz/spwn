package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
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
	Alive(pid int) bool
	Signal(pid int, sig Sig) error
	Reap(dir string) int
}

type FS interface {
	fs.FS
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
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
	// value: one UDP datagram broadcast on every LAN this Mac is on
	// (255.255.255.255), to port.
	Broadcast(ctx context.Context, port int, payload []byte) error
	// value: the hardware address of the interface that holds ip, as
	// aa:bb:cc:dd:ee:ff; empty when none does.
	HardwareAddress(ip string) string
}

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
