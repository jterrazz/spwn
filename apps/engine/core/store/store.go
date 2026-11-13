package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
)

var ErrNotFound = errors.New("no such record")

type UnreadableError struct {
	Path  string
	Cause error
	// value: the line of the file the cause is on; zero when it names none.
	Line int
}

func (e *UnreadableError) Error() string { return "cannot read " + e.Path + ": " + e.Cause.Error() }
func (e *UnreadableError) Unwrap() error { return e.Cause }

func LineOf(data []byte, err error) int {
	// value: the line of data a JSON syntax error is on, counted from 1; zero
	// for any other error.
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		return 0
	}
	return bytes.Count(data[:min(int(syntax.Offset), len(data))], []byte("\n")) + 1
}

type UnwritableError struct {
	Path  string
	Cause error
}

func (e *UnwritableError) Error() string { return "cannot write " + e.Path + ": " + e.Cause.Error() }
func (e *UnwritableError) Unwrap() error { return e.Cause }

type NewerError struct {
	Version, Known int
}

func (e *NewerError) Error() string {
	return "written at version " + strconv.Itoa(e.Version) + " by a newer station, this one knows " + strconv.Itoa(e.Known)
}

type OlderError struct {
	Version, Current int
}

func (e *OlderError) Error() string {
	return "written at version " + strconv.Itoa(e.Version) + " by an older station, this one reads " + strconv.Itoa(e.Current)
}

type Migrator interface {
	// invariant: brings every record of the store to its current version and
	// leaves a record already there byte for byte; run by setup's layout step
	// before any call reads the store. A record it cannot read is left as it
	// is, for the call that reads it to name; only a record a newer station
	// wrote stops it.
	Migrate(ctx context.Context) error
}

func OnlyNewer(err error) error {
	var newer *NewerError
	if errors.As(err, &newer) {
		return err
	}
	return nil
}

func Current(version, known int) (int, error) {
	// invariant: a record with no version field is at version 1, the shape
	// every store had before it carried one, so a store at version 1 writes none.
	if version == 0 {
		version = 1
	}
	if version > known {
		return version, &NewerError{Version: version, Known: known}
	}
	return version, nil
}
