package parser

import (
	"time"
)

// Parser parses a *RawResult into a user-defined structure.
type Parser interface {
	Parse(rawResult *RawResult, dst any) error
}

// RawResult captures the execution outcome
type RawResult struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Err      error
}
