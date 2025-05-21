package rexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/config"
	"github.com/ngrsoftlab/rexec/parser"
	"github.com/ngrsoftlab/rexec/utils"
)

// LocalSession execute commands on the local machine
type LocalSession struct {
	cfg    *config.LocalConfig
	mapper *utils.ExitCodeMapper
}

// NewLocalSession creates a LocalSession with the given LocalConfig
// If cfg==nil - config.NewLocalConfig() will be used
func NewLocalSession(cfg *config.LocalConfig) *LocalSession {
	if cfg == nil {
		cfg = config.NewLocalConfig()
	}
	return &LocalSession{cfg: cfg, mapper: utils.NewDefaultExitCodeMapper()}
}

// Run executes cmd and returns RawResult and an error. If cmd.Parser != nil && dst != nil, return parsed result in var
func (s *LocalSession) Run(ctx context.Context, cmd *command.Command, dst any, opts ...RunOption) (rawResult *parser.RawResult,
	err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic in Run: %v\n%s", r, debug.Stack())
		}
	}()

	rawResult = &parser.RawResult{}
	runCfg := newRunConfig(s.cfg.WorkDir, s.cfg.EnvVars, opts...)

	if validateErr := s.cfg.Validate(); validateErr != nil {
		return rawResult, fmt.Errorf("config is invalid: %w", validateErr)
	}

	execCmd := s.prepareCommandContext(ctx, cmd, runCfg)

	if runCaptureErr := s.runAndCapture(ctx, runCfg, execCmd, rawResult); runCaptureErr != nil {
		return rawResult, runCaptureErr
	}

	if parseErr := s.applyParser(rawResult, cmd, dst); parseErr != nil {
		return rawResult, parseErr
	}

	return rawResult, err
}

// Close does nothing for the local session.
func (s *LocalSession) Close() error {
	return nil
}

// prepareCommandContext builds *exec.Cmd for “sh -c <cmd>”,
// sets the working directory and environment from runConfig.
func (s *LocalSession) prepareCommandContext(ctx context.Context, cmd *command.Command, cfg *runConfig) *exec.Cmd {
	execCmd := exec.CommandContext(ctx, "sh", "-c", cmd.String())
	// change workdir
	execCmd.Dir = cfg.Dir

	// settle environment
	env := os.Environ()
	for k, v := range cfg.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	execCmd.Env = env

	return execCmd
}

// runAndCapture runs cmd.Run(), measures time, fills raw.Stdout/raw.Stderr,
func (s *LocalSession) runAndCapture(ctx context.Context, cfg *runConfig, c *exec.Cmd, rawResult *parser.RawResult) error {
	var outBuf, errBuf bytes.Buffer

	stdout := cfg.Stdout
	if stdout == nil {
		stdout = &outBuf
	}

	stderr := cfg.Stderr
	if stderr == nil {
		stderr = &errBuf
	}

	c.Stdout, c.Stderr = stdout, stderr

	start := time.Now()
	runErr := c.Run()
	rawResult.Duration = time.Since(start)

	if cfg.Stdout == nil {
		rawResult.Stdout = outBuf.String()
	}

	if cfg.Stderr == nil {
		rawResult.Stderr = errBuf.String()
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		err := fmt.Errorf(
			"command canceled after %s: %w",
			rawResult.Duration.Truncate(time.Millisecond),
			ctxErr,
		)
		rawResult.ExitCode = -1
		rawResult.Err = err
		return err
	}

	if runErr != nil {
		code := -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			code = exitErr.ExitCode()
		}
		msg := s.mapper.Lookup(code)
		stderrText := strings.TrimSpace(rawResult.Stderr)
		if len(stderrText) > 200 {
			stderrText = stderrText[:200] + "..."
		}
		err := fmt.Errorf("command failed (%s): %s: %w", msg, stderrText, runErr)
		rawResult.ExitCode = code
		rawResult.Err = err
		return err
	}

	rawResult.ExitCode = 0
	return nil
}

// applyParser calls cmd.Parser.Parse(raw, dst) if Parser != nil.
func (s *LocalSession) applyParser(result *parser.RawResult, cmd *command.Command, dst any) error {
	if cmd.Parser != nil && dst != nil {
		if parseErr := cmd.Parser.Parse(result, dst); parseErr != nil {
			return fmt.Errorf("parse error: %w", parseErr)
		}
	}
	return nil
}
