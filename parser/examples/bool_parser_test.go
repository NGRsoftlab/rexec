// Copyright © NGRSoftlab 2020-2025

package examples

import (
	"testing"

	"github.com/ngrsoftlab/rexec/parser"
)

func TestPathExistence_Parse(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"true_lower", "true", true},
		{"true_upper_space", " TRUE \n", true},
		{"false_lower", "false", false},
		{"text", "yes", true},
		{"upper_text", "F", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := &parser.RawResult{Stdout: tc.output}
			var got bool
			if err := (&BoolParser{}).Parse(raw, &got); err != nil {
				t.Fatalf("%s: unexpected error %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("%s: got %v; want %v", tc.name, got, tc.want)
			}
		})
	}
}
