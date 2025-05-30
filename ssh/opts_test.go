// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewRunConfig_TableDriven(t *testing.T) {
	env := map[string]string{"A": "1"}
	tests := []struct {
		name          string
		envVars       map[string]string
		opts          []RunOption
		wantEnv       map[string]string
		wantBufOutCap bool
		wantStream    bool
	}{
		{"default", env, nil, env, true, false},
		{"streaming", nil, []RunOption{WithStreaming()}, map[string]string{}, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc := newRunConfig("", tc.envVars, tc.opts...)

			if !reflect.DeepEqual(rc.env, tc.wantEnv) {
				t.Errorf("env = %v; want %v", rc.env, tc.wantEnv)
			}

			if rc.stream != tc.wantStream {
				t.Errorf("stream = %v; want %v", rc.stream, tc.wantStream)
			}

			rc.bufOut.Reset()
			n, _ := rc.stdout.Write([]byte("x"))
			if n != 1 {
				t.Fatalf("stdout write wrote %d; want 1", n)
			}
			got := rc.bufOut.Len() > 0
			if got != tc.wantBufOutCap {
				t.Errorf("bufOut captured = %v; want %v", got, tc.wantBufOutCap)
			}
		})
	}
}

func TestRunConfigOptions(t *testing.T) {
	in := bytes.NewBufferString("input")
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	tests := []struct {
		name      string
		op        RunOption
		execOn    *runConfig
		wantField string
		wantValue interface{}
	}{
		{"envvar", WithEnvVar("B", "2"), newRunConfig("", nil), "env[\"B\"]", "2"},
		{"stdin", WithStdin(in), newRunConfig("", nil), "stdin", in},
		{"stdout", WithStdout(out), newRunConfig("", nil), "stdout", out},
		{"stderr", WithStderr(errBuf), newRunConfig("", nil), "stderr", errBuf},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc := tc.execOn
			tc.op(rc)

			switch tc.wantField {
			case "env[\"B\"]":
				if rc.env["B"] != tc.wantValue {
					t.Errorf("env B = %v; want %v", rc.env["B"], tc.wantValue)
				}
			case "stdin":
				if rc.stdin != tc.wantValue {
					t.Errorf("stdin = %v; want %v", rc.stdin, tc.wantValue)
				}
			case "stdout":
				if rc.stdout != tc.wantValue {
					t.Errorf("stdout = %v; want %v", rc.stdout, tc.wantValue)
				}
			case "stderr":
				if rc.stderr != tc.wantValue {
					t.Errorf("stderr = %v; want %v", rc.stderr, tc.wantValue)
				}
			}
		})
	}
}
