package rexec

import (
	"context"
	"fmt"

	"github.com/ngrsoftlab/rexec/command"
)

// RunNoResult runs cmd and returns only an error (no stdout/stderr)
func RunNoResult[O any](ctx context.Context, sess Client[O], cmd *command.Command, opts ...O) error {
	if sess == nil {
		return fmt.Errorf("session is nil")
	}
	_, err := sess.Run(ctx, cmd, nil, opts...)
	return err
}

// RunRaw runs cmd and returns its stdout, stderr, exit code and error
func RunRaw[O any](ctx context.Context, sess Client[O], cmd *command.Command, opts ...O) (stdout string,
	stderr string, exitCode int, err error) {
	if sess == nil {
		return "", "", -1, fmt.Errorf("session is nil")
	}
	rr, err := sess.Run(ctx, cmd, nil, opts...)
	return rr.Stdout, rr.Stderr, rr.ExitCode, err
}

// RunParse [T] runs cmd, parses into a T, and returns the typed result
func RunParse[O, T any](ctx context.Context, sess Client[O], cmd *command.Command, opts ...O) (dst T, err error) {
	if sess == nil {
		return dst, fmt.Errorf("session is nil")
	}
	_, err = sess.Run(ctx, cmd, &dst, opts...)
	return dst, err
}
