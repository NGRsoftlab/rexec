package rexec

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/utils"
)

// RunNoResult runs cmd and returns only an error (no stdout/stderr)
func RunNoResult[O any](ctx context.Context, client Client[O], cmd *command.Command, opts ...O) error {
	if client == nil {
		return utils.ErrClientNil
	}
	_, err := client.Run(ctx, cmd, nil, opts...)
	return err
}

// RunRaw runs cmd and returns its stdout, stderr, exit code and error
func RunRaw[O any](ctx context.Context, client Client[O], cmd *command.Command, opts ...O) (stdout string,
	stderr string, exitCode int, err error) {
	if client == nil {
		return "", "", -1, utils.ErrClientNil
	}
	rr, err := client.Run(ctx, cmd, nil, opts...)
	return rr.Stdout, rr.Stderr, rr.ExitCode, err
}

// RunParse [T] runs cmd, parses into a T, and returns the typed result
func RunParse[O, T any](ctx context.Context, client Client[O], cmd *command.Command, opts ...O) (dst T, err error) {
	if client == nil {
		return dst, utils.ErrClientNil
	}
	_, err = client.Run(ctx, cmd, &dst, opts...)
	return dst, err
}
