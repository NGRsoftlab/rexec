package rexec

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
)

// Client - single interface for local and SSH session
// Run runs cmd, parses the result to dst (if Parser!=nil)
// and applies opts options.
type Client[O any] interface {
	Run(ctx context.Context, cmd *command.Command, dst any, opts ...O) (*parser.RawResult, error)
	Close() error
}
