package executor

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
)

// Executor is the abstraction over running a command.
type Executor interface {
	Run(ctx context.Context, cmd *command.Command) *parser.RawResult
}
