package local

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
)

func TestNewClientAndClose(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{"nil_cfg", nil},
		{"custom_cfg", &Config{WorkDir: ".", EnvVars: map[string]string{"A": "1"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cl := NewClient(tc.cfg)
			if tc.cfg == nil {
				if cl.cfg == nil || cl.cfg.WorkDir != "" {
					t.Errorf("NewClient(nil): cfg.WorkDir = %q; want empty", cl.cfg.WorkDir)
				}
			} else {
				if !reflect.DeepEqual(cl.cfg, tc.cfg) {
					t.Errorf("NewClient: cfg = %+v; want %+v", cl.cfg, tc.cfg)
				}
			}
			if err := cl.Close(); err != nil {
				t.Errorf("Close() error = %v; want nil", err)
			}
		})
	}
}

func TestPrepareCommandContext(t *testing.T) {
	cl := NewClient(nil)
	baseEnv := map[string]string{"X": "1"}
	tests := []struct {
		name    string
		workdir string
		envVars map[string]string
		cmd     *command.Command
	}{
		{"defaults", "", nil, command.New("echo hello")},
		{"custom", "/tmp", baseEnv, command.New("echo %s", command.WithArgs("hi"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runc := newRunConfig(tc.workdir, tc.envVars)
			execCmd := cl.prepareCommandContext(context.Background(), tc.cmd, runc)
			if len(execCmd.Args) < 3 || execCmd.Args[0] != "sh" || execCmd.Args[1] != "-c" || execCmd.Args[2] != tc.cmd.String() {
				t.Errorf("Args = %v; want [sh -c, %q]", execCmd.Args, tc.cmd.String())
			}
			if execCmd.Dir != tc.workdir {
				t.Errorf("Dir = %q; want %q", execCmd.Dir, tc.workdir)
			}
			if tc.envVars != nil {
				found := false
				for _, e := range execCmd.Env {
					if strings.HasPrefix(e, "X=") {
						found = true
					}
				}
				if !found {
					t.Errorf("Env = %v; want include X=...", execCmd.Env)
				}
			}
		})
	}
}

func TestRunAndCapture(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found in PATH, skipping")
	}

	cl := NewClient(nil)
	cfg := newRunConfig("", nil)
	tests := []struct {
		name       string
		commands   []string
		wantErr    bool
		wantStdout string
		wantCode   int
	}{
		{"success_true", []string{"true"}, false, "", 0},
		{"success_echo", []string{"echo -n rexec"}, false, "rexec", 0},
		{"fail_false", []string{"false"}, true, "", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(context.Background(), "sh", "-c", strings.Join(tc.commands, " "))
			rr := parser.NewRawResult(strings.Join(tc.commands, " "))
			err := cl.runAndCapture(context.Background(), cfg, cmd, rr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v; wantErr %v", err, tc.wantErr)
			}
			if rr.ExitCode != tc.wantCode {
				t.Errorf("ExitCode = %d; want %d", rr.ExitCode, tc.wantCode)
			}
			if tc.wantStdout != "" && rr.Stdout != tc.wantStdout {
				t.Errorf("Stdout = %q; want %q", rr.Stdout, tc.wantStdout)
			}
			if rr.Duration < 0 {
				t.Errorf("Duration = %v; want >=0", rr.Duration)
			}
		})
	}
}

type nopParser struct{}

func (nopParser) Parse(raw *parser.RawResult, dst any) error { return nil }

type errParser struct{}

func (errParser) Parse(raw *parser.RawResult, dst any) error { return errors.New("parse failed") }

func TestApplyParser(t *testing.T) {
	cl := NewClient(nil)
	tests := []struct {
		name    string
		rr      *parser.RawResult
		cmd     *command.Command
		dst     *int
		wantErr bool
	}{
		{"no_parser", parser.NewRawResult(""), command.New("echo"), new(int), false},
		{"nil_dst", parser.NewRawResult(""), &command.Command{Template: "", Parser: nopParser{}}, nil, false},
		{"parser_error", parser.NewRawResult(""), &command.Command{Template: "", Parser: errParser{}}, new(int), true},
		{"parser_success", parser.NewRawResult(""), &command.Command{Template: "", Parser: nopParser{}}, new(int), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cl.applyParser(tc.rr, tc.cmd, tc.dst)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tc.wantErr)
			}
		})
	}
}
