package config

import (
	"fmt"
	"os"
)

// LocalConfig holds settings for a local session
type LocalConfig struct {
	WorkDir string            // optional: working directory for the command
	EnvVars map[string]string // optional: extra environment variables
}

// NewLocalConfig creates a LocalConfig
func NewLocalConfig() *LocalConfig {
	return &LocalConfig{
		WorkDir: "",
		EnvVars: make(map[string]string),
	}
}

// WithWorkDir sets the working directory for the command
func (lc *LocalConfig) WithWorkDir(workdir string) *LocalConfig {
	if workdir != "" {
		lc.WorkDir = workdir
	}
	return lc
}

// WithEnvVars merges in extra environment variables.
func (lc *LocalConfig) WithEnvVars(env map[string]string) *LocalConfig {
	for k, v := range env {
		lc.EnvVars[k] = v
	}
	return lc
}

func (lc *LocalConfig) Validate() error {
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
