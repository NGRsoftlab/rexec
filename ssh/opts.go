package ssh

import (
	"bytes"
	"io"
	"sync"
)

// RunOption define command run options
type RunOption func(*runConfig)

var bufPoolOut = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

var bufPoolErr = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

type runConfig struct {
	env           map[string]string // additional remove env vars
	stdin         io.Reader         // optional stdin for the command
	stdout        io.Writer         // optional stdin for the command
	stderr        io.Writer         // optional stdin for the command
	bufOut        *bytes.Buffer
	bufErr        *bytes.Buffer
	usePTY        bool
	stream        bool
	disableBuffer bool
}

// newRunConfig - create new default run config for command
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

// WithEnvVar adds an env var for this run
func WithEnvVar(key, value string) RunOption {
	return func(config *runConfig) {
		config.env[key] = value
	}
}

// WithStdin set custom io.Reader for input
func WithStdin(stdin io.Reader) RunOption {
	return func(config *runConfig) {
		config.stdin = stdin
	}
}

// WithStdout set custom io.Writer for output
func WithStdout(stdout io.Writer) RunOption {
	return func(config *runConfig) {
		config.stdout = stdout
	}
}

// WithStderr set custom io.Writer for output
func WithStderr(stderr io.Writer) RunOption {
	return func(config *runConfig) {
		config.stderr = stderr
	}
}

// WithStreaming set steam flag on true, and give possibility take output on realtime. Write on custom writer and buf.
func WithStreaming() RunOption {
	return func(config *runConfig) {
		config.stream = true
	}
}

func WithoutBuffering() RunOption {
	return func(config *runConfig) {
		config.disableBuffer = true
	}
}
