package config

import "time"

// UniverseConfig holds the configuration for creating a universe.
type UniverseConfig struct {
	Image     string
	Mind      string
	Workspace string
	Memory    string
	CPU       float64
	Timeout   time.Duration
}

// Universe represents a running or stopped universe.
type Universe struct {
	ID        string
	Status    UniverseStatus
	Image     string
	Mind      string
	CreatedAt time.Time
}

// UniverseStatus represents the state of a universe.
type UniverseStatus string

const (
	StatusCreated UniverseStatus = "created"
	StatusRunning UniverseStatus = "running"
	StatusStopped UniverseStatus = "stopped"
)
