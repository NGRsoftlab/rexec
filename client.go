package rexec

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
)

// Client - single interface for local and SSH session
// Run runs cmd, parses the result to dst (if Parser!=nil)
// and applies opts options to this run only.
type Client[O any] interface {
	Run(ctx context.Context, cmd *command.Command, dst any, opts ...O) (*parser.RawResult, error)
	Close() error
	//  ExecuteAndParse(ctx context.Context, cmd *command.Command, out any) error
	//	 TransferFile(ctx context.Context, src io.Reader, destPath string, mode os.FileMode) error
	// RunIgnoreErrors(ctx context.Context, cmd string) error
	// RunWithOutput(ctx context.Context, cmd string) (string, error)
	// RunAndParse(ctx context.Context, cmd string, parser parser.Parser) error
}
