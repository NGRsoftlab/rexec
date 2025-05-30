// Copyright © NGRSoftlab 2020-2025

package local

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestNewRunConfig(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	tests := []struct {
		name       string
		baseDir    string
		baseEnv    map[string]string
		opts       []RunOption
		wantDir    string
		wantEnv    map[string]string
		wantStdout io.Writer
		wantStderr io.Writer
	}{
		{
			name:    "no_options",
			baseDir: "/base", baseEnv: map[string]string{"A": "1"},
			opts:    nil,
			wantDir: "/base", wantEnv: map[string]string{"A": "1"},
			wantStdout: nil, wantStderr: nil,
		},
		{
			name:    "workdir_override",
			baseDir: "/base", baseEnv: map[string]string{},
			opts:    []RunOption{WithWorkdir("/new")},
			wantDir: "/new", wantEnv: map[string]string{},
			wantStdout: nil, wantStderr: nil,
		},
		{
			name:    "env_override",
			baseDir: ".", baseEnv: map[string]string{"X": "old"},
			opts:    []RunOption{WithEnvVar("X", "new"), WithEnvVar("Y", "yval")},
			wantDir: ".", wantEnv: map[string]string{"X": "new", "Y": "yval"},
			wantStdout: nil, wantStderr: nil,
		},
		{
			name:    "stdout_stderr",
			baseDir: ".", baseEnv: map[string]string{},
			opts:    []RunOption{WithStdout(bufOut), WithStderr(bufErr)},
			wantDir: ".", wantEnv: map[string]string{},
			wantStdout: bufOut, wantStderr: bufErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newRunConfig(tc.baseDir, tc.baseEnv, tc.opts...)

			if cfg.dir != tc.wantDir {
				t.Errorf("dir = %q; want %q", cfg.dir, tc.wantDir)
			}

			if !reflect.DeepEqual(cfg.envVars, tc.wantEnv) {
				t.Errorf("envVars = %#v; want %#v", cfg.envVars, tc.wantEnv)
			}

			if cfg.stdout != tc.wantStdout {
				t.Errorf("stdout writer = %v; want %v", cfg.stdout, tc.wantStdout)
			}

			if cfg.stderr != tc.wantStderr {
				t.Errorf("stderr writer = %v; want %v", cfg.stderr, tc.wantStderr)
			}
		})
	}
}
