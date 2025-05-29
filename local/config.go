// Copyright © NGRSoftlab 2020-2025

package local

import (
	"fmt"
	"os"
)

// Config holds settings for running commands locally
type Config struct {
	WorkDir string            // directory in which to execute commands
	EnvVars map[string]string // additional environment variables to set
}

// NewConfig creates a Config with defaults (no workdir, empty env)
func NewConfig() *Config {
	return &Config{
		WorkDir: "",
		EnvVars: make(map[string]string),
	}
}

// WithWorkDir sets the working directory if non-empty
func (lc *Config) WithWorkDir(workdir string) *Config {
	if workdir != "" {
		lc.WorkDir = workdir
	}
	return lc
}

// WithEnvVars adds or overrides environment variables
func (lc *Config) WithEnvVars(env map[string]string) *Config {
	for k, v := range env {
		lc.EnvVars[k] = v
	}
	return lc
}

// Validate checks that WorkDir exists and is a directory
func (lc *Config) Validate() error {
	if lc.WorkDir == "" {
		return nil
	}
	fi, err := os.Stat(lc.WorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workdir %s does not exist", lc.WorkDir)
		}
		return fmt.Errorf("invalid workdir %q: %w", lc.WorkDir, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("workdir %q is not a directory", lc.WorkDir)
	}
	return nil
}
