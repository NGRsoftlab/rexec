// Copyright © NGRSoftlab 2020-2025

package parser

import (
	"time"
)

// Parser converts a RawResult into a user-defined value
type Parser interface {
	Parse(rawResult *RawResult, dst any) error
}

// RawResult holds the outcome of running a command
type RawResult struct {
	Command  string        // the exact command string executed
	Stdout   string        // collected standard output
	Stderr   string        // collected standard error
	ExitCode int           // process exit code
	Duration time.Duration // time taken to run the command
	Err      error         // any error from execution or parsing
}

// NewRawResult initializes a RawResult for the given shell command
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
