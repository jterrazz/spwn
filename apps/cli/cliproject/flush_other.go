//go:build !linux && !darwin

package cliproject

func flushInput(int) error { return nil }
