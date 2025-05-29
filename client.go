package rexec

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
)

// Client represents an execution engine (local or SSH) that can run commands.
// The type parameter O specifies the kind of options the client accepts.
type Client[O any] interface {
	// Run executes the given Command, applies any provided options,
	// and, if a Parser is set on cmd, parses the result into dst.
	// Returns a RawResult with stdout, stderr, exit code, and timing.
	Run(ctx context.Context, cmd *command.Command, dst any, opts ...O) (*parser.RawResult, error)

	// Close releases resources held by the client (e.g., SSH sessions).
	// For local clients this is a no-op.
	Close() error
}
