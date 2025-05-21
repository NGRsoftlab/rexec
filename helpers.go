package rexec

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
)

// RunNoResult runs cmd and returns only an error (no stdout/stderr)
func RunNoResult(ctx context.Context, sess Session, cmd *command.Command, opts ...RunOption) error {
	_, err := sess.Run(ctx, cmd, nil, opts...)
	return err
}

// RunRaw runs cmd and returns its stdout, stderr, exitcode and error
func RunRaw(ctx context.Context, sess Session, cmd *command.Command, opts ...RunOption) (stdout string, stderr string, exitCode int, err error) {
	rr, err := sess.Run(ctx, cmd, nil, opts...)
	return rr.Stdout, rr.Stderr, rr.ExitCode, err
}

// RunParse [T] runs cmd, parses into a T, and returns the typed result
func RunParse[T any](ctx context.Context, sess Session, cmd *command.Command, opts ...RunOption) (dst T, err error) {
	_, err = sess.Run(ctx, cmd, &dst, opts...)
	return dst, err
}
