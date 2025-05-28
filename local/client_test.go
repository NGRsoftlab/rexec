package local_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/local"
)

func TestLocalSession_Run(t *testing.T) {
	tmp := t.TempDir()
	noExecFile := filepath.Join(tmp, "noexec.sh")
	if err := os.WriteFile(noExecFile, []byte("noexec"), 0400); err != nil {
		t.Fatal(err)
	}

	sess := local.NewSession(local.NewConfig().WithWorkDir(tmp))

	tests := []struct {
		name       string
		cmd        *command.Command
		wantCode   int
		wantErrSub string
	}{
		{"success_exit0", command.New("echo %s", command.WithArgs("ok")), 0, ""},
		{"general_error_exit1", command.New("false"), 1, "general error"},
		{"custom_exit5", command.New("%s", command.WithArgs("exit 5")), 5, "exit 5"},
		{"not_found_exit127", command.New("no_such_cmd"), 127, "command not found"},
		{"permission_denied_exit126", command.New("%s", command.WithArgs(noExecFile)), 126,
			"permission denied"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr, err := sess.Run(context.Background(), tc.cmd, nil)
			if rr.ExitCode != tc.wantCode {
				t.Errorf("%s: got code %d; want %d", tc.name, rr.ExitCode, tc.wantCode)
			}
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("%s: expected error containing %q, got nil", tc.name, tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Errorf("%s: error %q does not contain %q", tc.name, err.Error(), tc.wantErrSub)
				}
			} else if err != nil {
				t.Errorf("%s: unexpected error: %v", tc.name, err)
			}
		})
	}
}
