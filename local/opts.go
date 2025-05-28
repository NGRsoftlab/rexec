package local

import (
	"io"
)

// RunOption - option that changes localRunConfig only for one call to Run
type RunOption func(*localRunConfig)

// localRunConfig collects startup parameters (working directory + environment variables)
type localRunConfig struct {
	dir     string
	envVars map[string]string
	stdout  io.Writer
	stderr  io.Writer
}

func newRunConfig(baseDir string, baseEnv map[string]string, opts ...RunOption) *localRunConfig {
	runConfig := &localRunConfig{
		dir:     baseDir,
		envVars: make(map[string]string, len(baseEnv)),
	}

	for k, v := range baseEnv {
		runConfig.envVars[k] = v
	}

	for _, opt := range opts {
		opt(runConfig)
	}
	return runConfig
}

// WithWorkdir sets the working directory for one run
func WithWorkdir(workdir string) RunOption {
	return func(rc *localRunConfig) {
		rc.dir = workdir
	}
}

// WithEnvVar adds or overrides one environment variable for a one run
func WithEnvVar(key, value string) RunOption {
	return func(rc *localRunConfig) {
		rc.envVars[key] = value
	}
}

// WithStdout sends live stdout to w instead of buffering.
func WithStdout(stdout io.Writer) RunOption {
	return func(rc *localRunConfig) {
		rc.stdout = stdout
	}
}

// WithStderr sends live stderr to w instead of buffering.
func WithStderr(stderr io.Writer) RunOption {
	return func(rc *localRunConfig) {
		rc.stderr = stderr
	}
}
