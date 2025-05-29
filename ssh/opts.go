package ssh

import (
	"bytes"
	"io"
	"sync"
)

// RunOption configures a single SSH command execution
type RunOption func(*runConfig)

// bufPoolOut is a pool of buffers used to capture stdout
var bufPoolOut = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// bufPoolErr is a pool of buffers used to capture stderr
var bufPoolErr = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// runConfig holds settings and buffers for one SSH command run
type runConfig struct {
	env           map[string]string // environment variables for this run
	stdin         io.Reader         // input for the command
	stdout        io.Writer         // live stdout writer (wrapped to buffer by default)
	stderr        io.Writer         // live stderr writer (wrapped to buffer by default)
	bufOut        *bytes.Buffer     // internal buffer for stdout
	bufErr        *bytes.Buffer     // internal buffer for stderr
	usePTY        bool              // allocate a PTY for the session
	stream        bool              // stream output in real time
	disableBuffer bool              // disable internal buffering of output
}

// newRunConfig creates a runConfig from base envVars and applies opts.
// It initializes internal buffers and, unless buffering is disabled,
// wraps stdout/stderr writers to also record output in bufOut/bufErr
func newRunConfig(workDir string, envVars map[string]string, opts ...RunOption) *runConfig {
	bufOut := bufPoolOut.Get().(*bytes.Buffer)
	bufErr := bufPoolErr.Get().(*bytes.Buffer)
	bufOut.Reset()
	bufErr.Reset()

	runConfig := &runConfig{
		env:    make(map[string]string, len(envVars)),
		stdout: bufOut, // default buffers
		stderr: bufErr,
		bufOut: bufOut,
		bufErr: bufErr,
		stream: false,
	}

	for k, v := range envVars {
		runConfig.env[k] = v
	}

	for _, opt := range opts {
		opt(runConfig)
	}

	if !runConfig.disableBuffer {
		if runConfig.stdout != bufOut {
			runConfig.stdout = io.MultiWriter(runConfig.stdout, bufOut)
		}
		if runConfig.stderr != bufErr {
			runConfig.stderr = io.MultiWriter(runConfig.stderr, bufErr)
		}
	}
	return runConfig
}

// WithEnvVar adds or overrides an environment variable for this run
func WithEnvVar(key, value string) RunOption {
	return func(config *runConfig) {
		config.env[key] = value
	}
}

// WithStdin sets the input reader for the command
func WithStdin(stdin io.Reader) RunOption {
	return func(config *runConfig) {
		config.stdin = stdin
	}
}

// WithStdout sets a custom writer for live stdout
func WithStdout(stdout io.Writer) RunOption {
	return func(config *runConfig) {
		config.stdout = stdout
	}
}

// WithStderr sets a custom writer for live stderr
func WithStderr(stderr io.Writer) RunOption {
	return func(config *runConfig) {
		config.stderr = stderr
	}
}

// WithStreaming enables real-time streaming of stdout and stderr as data arrives
func WithStreaming() RunOption {
	return func(config *runConfig) {
		config.stream = true
	}
}

// WithoutBuffering disables internal buffering of output, so only provided stdout/stderr writers receive data
func WithoutBuffering() RunOption {
	return func(config *runConfig) {
		config.disableBuffer = true
	}
}
