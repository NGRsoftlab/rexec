// Copyright © NGRSoftlab 2020-2025

package rexec

import (
	"context"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/utils"
)

// RunNoResult executes cmd using client, ignoring stdout/stderr.
// Returns any execution error
func RunNoResult[O any](ctx context.Context, client Client[O], cmd *command.Command, opts ...O) error {
	if client == nil {
		return utils.ErrClientNil
	}
	_, err := client.Run(ctx, cmd, nil, opts...)
	return err
}

// RunRaw executes cmd and returns its stdout, stderr, exit code, and error
func RunRaw[O any](ctx context.Context, client Client[O], cmd *command.Command, opts ...O) (stdout string,
	stderr string, exitCode int, err error) {
	if client == nil {
		return "", "", -1, utils.ErrClientNil
	}
	rr, err := client.Run(ctx, cmd, nil, opts...)
	if rr == nil {
		return "", "", -1, err
	}
	return rr.Stdout, rr.Stderr, rr.ExitCode, err
}

// RunParse executes cmd, parses its output into dst of type T, and returns dst and any error
func RunParse[O, T any](ctx context.Context, client Client[O], cmd *command.Command, opts ...O) (dst T, err error) {
	if client == nil {
		return dst, utils.ErrClientNil
	}
	_, err = client.Run(ctx, cmd, &dst, opts...)
	return dst, err
}
