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

func NewRawResult(shellCmd string) *RawResult {
	return &RawResult{
		Command:  shellCmd,
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
		Duration: 0,
		Err:      nil,
	}
}
