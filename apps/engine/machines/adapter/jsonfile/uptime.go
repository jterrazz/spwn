package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const (
	UptimeSchema  = "station.uptime"
	UptimeVersion = 1
)

type UptimeFiles interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
}

type Uptime struct {
	files UptimeFiles
	file  string
}

var (
	_ contract.UptimeStore = (*Uptime)(nil)
	_ store.Migrator       = (*Uptime)(nil)
)

func NewUptime(files UptimeFiles, file string) *Uptime { return &Uptime{files: files, file: file} }

// invariant: the file is {"schema": "station.uptime/v<N>", "servers": {name: record}}:
// the schema is its version, and a file with none is at version 1.
type uptimeFile struct {
	Schema  string                     `json:"schema"`
	Servers map[string]machines.Uptime `json:"servers"`
}

func (u *Uptime) load() (map[string]machines.Uptime, error) {
	data, err := u.files.ReadFile(u.file)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]machines.Uptime{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f uptimeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, &store.UnreadableError{Path: u.file, Cause: err}
	}
	_, version, _ := strings.Cut(f.Schema, "/v")
	n, _ := strconv.Atoi(version)
	if _, err := store.Current(n, UptimeVersion); err != nil {
		return nil, &store.UnreadableError{Path: u.file, Cause: err}
	}
	if f.Servers == nil {
		f.Servers = map[string]machines.Uptime{}
	}
	return f.Servers, nil
}

// invariant: the file is written whole to a temporary file then renamed, so
// a reader (the CLI while the engine marks) never sees half a file.
func (u *Uptime) save(servers map[string]machines.Uptime) error {
	data, err := json.MarshalIndent(uptimeFile{Schema: UptimeSchema + "/v" + strconv.Itoa(UptimeVersion), Servers: servers}, "", "  ")
	if err != nil {
		return err
	}
	if err := u.files.MkdirAll(path.Dir(u.file), 0o700); err != nil {
		return err
	}
	tmp := u.file + ".tmp"
	if err := u.files.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return u.files.Rename(tmp, u.file)
}

func (u *Uptime) Get(_ context.Context, server string) (machines.Uptime, error) {
	servers, err := u.load()
	if err != nil {
		return machines.Uptime{}, err
	}
	return servers[server], nil
}

func (u *Uptime) Put(_ context.Context, server string, record machines.Uptime) error {
	servers, err := u.load()
	if err != nil {
		return err
	}
	servers[server] = record
	return u.save(servers)
}

func (u *Uptime) Delete(_ context.Context, server string) error {
	servers, err := u.load()
	if err != nil {
		return err
	}
	if _, ok := servers[server]; !ok {
		return nil
	}
	delete(servers, server)
	return u.save(servers)
}

func (u *Uptime) Migrate(context.Context) error {
	_, err := u.load()
	return store.OnlyNewer(err)
}
