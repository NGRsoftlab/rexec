package local

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

	"github.com/ngrsoftlab/rexec"
	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
	"github.com/ngrsoftlab/rexec/utils"
)

// interface guard
var _ rexec.Client[RunOption] = (*Client)(nil)

// Client execute commands on the local machine
type Client struct {
	cfg    *Config
	mapper *utils.ExitCodeMapper
}

// NewSession creates a LocalSession with the given config
// If cfg==nil - config.NewConfig() will be used
func NewSession(cfg *Config) *Client {
	if cfg == nil {
		cfg = NewConfig()
	}
	return &Client{cfg: cfg, mapper: utils.NewDefaultExitCodeMapper()}
}

// Run executes cmd and returns RawResult and an error. If cmd.Parser != nil && dst != nil, return parsed result in var
func (cl *Client) Run(ctx context.Context, cmd *command.Command, dst any, opts ...RunOption) (*parser.RawResult, error) {
	var err error
	shellCmd := cmd.String()
	result := parser.NewRawResult(shellCmd)

	defer func() {
		if r := recover(); r != nil {
			result.Err = fmt.Errorf("recovered from panic on run: %v\n%s", r, debug.Stack())
			result.ExitCode = -1
			err = result.Err
		}
	}()

	runCfg := newRunConfig(cl.cfg.WorkDir, cl.cfg.EnvVars, opts...)

	if validateErr := cl.cfg.Validate(); validateErr != nil {
		return result, fmt.Errorf("config is invalid: %w", validateErr)
	}

	execCmd := cl.prepareCommandContext(ctx, cmd, runCfg)

	if runCaptureErr := cl.runAndCapture(ctx, runCfg, execCmd, result); runCaptureErr != nil {
		return result, runCaptureErr
	}

	if parseErr := cl.applyParser(result, cmd, dst); parseErr != nil {
		return result, parseErr
	}

	return result, err
}

// Close does nothing for the local session.
func (cl *Client) Close() error {
	return nil
}

// prepareCommandContext builds *exec.Cmd for “sh -c <cmd>”,
// sets the working directory and environment from localRunConfig.
func (cl *Client) prepareCommandContext(ctx context.Context, cmd *command.Command, cfg *localRunConfig) *exec.Cmd {
	execCmd := exec.CommandContext(ctx, "sh", "-c", cmd.String())
	// change workdir
	execCmd.Dir = cfg.dir

	// settle environment
	env := os.Environ()
	for k, v := range cfg.envVars {
		env = append(env, fmt.Sprintf("%cl=%cl", k, v))
	}
	execCmd.Env = env

	return execCmd
}

// runAndCapture runs cmd.Run(), measures time, fills raw.stdout/raw.stderr,
func (cl *Client) runAndCapture(ctx context.Context, cfg *localRunConfig, c *exec.Cmd, rawResult *parser.RawResult) error {
	var outBuf, errBuf bytes.Buffer

	stdout := cfg.stdout
	if stdout == nil {
		stdout = &outBuf
	}

	stderr := cfg.stderr
	if stderr == nil {
		stderr = &errBuf
	}

	c.Stdout, c.Stderr = stdout, stderr

	start := time.Now()
	runErr := c.Run()
	rawResult.Duration = time.Since(start)

	if cfg.stdout == nil {
		rawResult.Stdout = outBuf.String()
	}

	if cfg.stderr == nil {
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
		msg := cl.mapper.Lookup(code)
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
func (cl *Client) applyParser(result *parser.RawResult, cmd *command.Command, dst any) error {
	if cmd.Parser != nil && dst != nil {
		if parseErr := cmd.Parser.Parse(result, dst); parseErr != nil {
			return fmt.Errorf("parse error: %w", parseErr)
		}
	}
	return nil
}
