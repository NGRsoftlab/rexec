// Copyright © NGRSoftlab 2020-2025

package local

import (
	"io"
)

// RunOption modifies settings for a single Run invocation
type RunOption func(*localRunConfig)

// localRunConfig holds per-call execution settings
type localRunConfig struct {
	dir     string            // working directory for this run
	envVars map[string]string // environment variables for this run
	stdout  io.Writer         // custom stdout writer (nil => buffer)
	stderr  io.Writer         // custom stderr writer (nil => buffer)
}

// newRunConfig creates a localRunConfig from base settings and applies opts
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

// WithWorkdir sets a custom working directory for one Run
func WithWorkdir(workdir string) RunOption {
	return func(rc *localRunConfig) {
		rc.dir = workdir
	}
}

// WithEnvVar adds or overrides one environment variable for a single Run
func WithEnvVar(key, value string) RunOption {
	return func(rc *localRunConfig) {
		rc.envVars[key] = value
	}
}

// WithStdout directs live stdout to the given writer instead of buffering
func WithStdout(stdout io.Writer) RunOption {
	return func(rc *localRunConfig) {
		rc.stdout = stdout
	}
}

// WithStderr directs live stderr to the given writer instead of buffering
func WithStderr(stderr io.Writer) RunOption {
	return func(rc *localRunConfig) {
		rc.stderr = stderr
	}
}
