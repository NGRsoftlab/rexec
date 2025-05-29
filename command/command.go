// Copyright © NGRSoftlab 2020-2025

package command

import (
	"fmt"

	"github.com/ngrsoftlab/rexec/parser"
)

// CmdOption defines a function that applies configuration to a Command
type CmdOption func(*Command)

// Command represents a shell command with a template, positional arguments, and an optional parser
type Command struct {
	Template string        // format string for the command, used with fmt.Sprintf
	Args     []any         // values to plug into the template
	Parser   parser.Parser // optional parser to process command output
}

// New returns a Command initialized with the given template and applies any CmdOption to it
func New(template string, opts ...CmdOption) *Command {
	c := &Command{Template: template}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithArgs returns a CmdOption that appends positional args for formatting the command template
func WithArgs(args ...any) CmdOption {
	return func(c *Command) {
		c.Args = append(c.Args, args...)
	}
}

// WithParser returns a CmdOption that assigns a parser to handle the raw command result
func WithParser(p parser.Parser) CmdOption {
	return func(c *Command) {
		c.Parser = p
	}
}

// String builds the final shell command by applying the template to its arguments
func (c *Command) String() string {
	return fmt.Sprintf(c.Template, c.Args...)
}
