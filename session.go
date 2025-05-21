package rexec

import (
	"context"
	"io"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
)

// RunOption - option that changes runConfig only for one call to Run
type RunOption func(*runConfig)

// runConfig collects startup parameters (working directory + environment variables)
type runConfig struct {
	Dir     string
	EnvVars map[string]string
	Stdout  io.Writer
	Stderr  io.Writer
}

func newRunConfig(baseDir string, baseEnv map[string]string, opts ...RunOption) *runConfig {
	rc := &runConfig{
		Dir:     baseDir,
		EnvVars: make(map[string]string, len(baseEnv)),
	}

	for k, v := range baseEnv {
		rc.EnvVars[k] = v
	}

	for _, o := range opts {
		o(rc)
	}
	return rc
}

// WithWorkdir sets the working directory for one run
func WithWorkdir(workdir string) RunOption {
	return func(rc *runConfig) {
		rc.Dir = workdir
	}
}

// WithEnvVar adds or overrides one environment variable for a one run
func WithEnvVar(key, value string) RunOption {
	return func(rc *runConfig) {
		rc.EnvVars[key] = value
	}
}

// WithStdout sends live stdout to w instead of buffering.
func WithStdout(stdout io.Writer) RunOption {
	return func(rc *runConfig) {
		rc.Stdout = stdout
	}
}

// WithStderr sends live stderr to w instead of buffering.
func WithStderr(stderr io.Writer) RunOption {
	return func(rc *runConfig) {
		rc.Stderr = stderr
	}
}

// Session - single interface for local and SSH session
// Run runs cmd, parses the result to dst (if Parser!=nil)
// and applies opts options to this run only.
type Session interface {
	Run(ctx context.Context, cmd *command.Command, dst any, opts ...RunOption) (*parser.RawResult, error)
	Close() error
	//  ExecuteAndParse(ctx context.Context, cmd *command.Command, out any) error
	//	 TransferFile(ctx context.Context, src io.Reader, destPath string, mode os.FileMode) error
	// RunIgnoreErrors(ctx context.Context, cmd string) error
	// RunWithOutput(ctx context.Context, cmd string) (string, error)
	// RunAndParse(ctx context.Context, cmd string, parser parser.Parser) error
}
