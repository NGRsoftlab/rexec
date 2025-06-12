// Copyright © NGRSoftlab 2020-2025

package rexec

import (
	"context"
	"fmt"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
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

// ParseWithMapping run the registered Parser for each executed command and
// store the parsed output into your destination variables.
// Only commands listed in dstMap will be parsed.
func ParseWithMapping(results map[*command.Command]*parser.RawResult, dstMap map[*command.Command]any) error {
	for cmd, dst := range dstMap {
		if dst == nil {
			return fmt.Errorf("dst must be non-nil pointer")
		}
		rawResult, ok := results[cmd]
		if !ok || rawResult == nil {
			continue
		}

		if cmd.Parser == nil {
			return fmt.Errorf("dst is set, but parser is nil for cmd %q", cmd.String())
		}

		if err := cmd.Parser.Parse(rawResult, dst); err != nil {
			return fmt.Errorf("parser failed for cmd %q: %w", cmd.String(), err)
		}
	}
	return nil
}

// ApplyParsers builds a temporary map from *command.Command to RawResult based on the
// supplied slice, then invokes ParseWithMapping for the entries in dstMap.
// Use this when RawResults are available as a slice, and you wish to parse only
// the commands specified in dstMap without manually creating the map yourself.
func ApplyParsers(results []*parser.RawResult, dstMap map[*command.Command]any) error {
	rawMap := make(map[*command.Command]*parser.RawResult, len(results))
	for _, rr := range results {
		if rr != nil {
			if cmd, ok := rr.CmdPtr.(*command.Command); ok {
				rawMap[cmd] = rr
			}
		}
	}
	return ParseWithMapping(rawMap, dstMap)
}
