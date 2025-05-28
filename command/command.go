package command

import (
	"fmt"

	"github.com/ngrsoftlab/rexec/parser"
)

// CmdOption configures a Command
type CmdOption func(*Command)

// Command represents a shell command to execute
type Command struct {
	Template string        // template of command
	Args     []any         // positional args for template
	Parser   parser.Parser // optional parser for command
}

// New creates a Command by applying any number of CommandOption.
// All parameters—including args—are set via options.
func New(template string, opts ...CmdOption) *Command {
	c := &Command{Template: template}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithArgs sets the positional arguments for fmt.Sprintf
func WithArgs(args ...any) CmdOption {
	return func(c *Command) {
		c.Args = append(c.Args, args...)
	}
}

// WithParser attaches a Parser that will be run after execution
func WithParser(p parser.Parser) CmdOption {
	return func(c *Command) {
		c.Parser = p
	}
}

// String renders the final shell command by applying arguments
func (c *Command) String() string {
	return fmt.Sprintf(c.Template, c.Args...)
}
