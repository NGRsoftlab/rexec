// Copyright © NGRSoftlab 2020-2025

package utils_test

import (
	"testing"

	"github.com/ngrsoftlab/rexec/utils"
)

func TestExitCodeMapper_Lookup(t *testing.T) {
	mapper := utils.NewDefaultExitCodeMapper()
	tests := []struct {
		code int
		want string
	}{
		{1, "general error"},
		{2, "invalid usage of shell builtins"},
		{64, "command line usage error"},
		{65, "data format error"},
		{126, "permission denied (cannot execute)"},
		{127, "command not found"},
		{128, "invalid exit argument"},
		{130, "script terminated by Control-C"},
		{137, "process killed (SIGKILL)"},
		{139, "segmentation fault (SIGSEGV)"},
		{143, "terminated by signal (SIGTERM)"},
		{131, "killed by signal 3"},
		{5, "exit 5"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := mapper.Lookup(tc.code)
			if got != tc.want {
				t.Errorf("Lookup(%d) = %q; want %q", tc.code, got, tc.want)
			}
		})
	}
}
