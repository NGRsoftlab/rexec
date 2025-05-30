// Copyright © NGRSoftlab 2020-2025

package command

import (
	"reflect"
	"testing"

	"github.com/ngrsoftlab/rexec/parser"
)

type nopParser struct{}

func (nopParser) Parse(raw *parser.RawResult, dst any) error { return nil }

func TestNew(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		opts          []CmdOption
		wantArgs      []any
		wantParserNil bool
	}{
		{"template_no_args", "echo", nil, nil, true},
		{"with_arg", "echo %s", []CmdOption{WithArgs("hello")}, []any{"hello"}, true},
		{"multi_args", "fmt %s %d", []CmdOption{
			WithArgs("first", 1),
			WithArgs("second", 2),
		}, []any{"first", 1, "second", 2}, true},
		{"with_parser", "tmpl", []CmdOption{WithParser(nopParser{})}, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.template, tc.opts...)
			if c.Template != tc.template {
				t.Errorf("%s: Template = %q; want %q", tc.name, c.Template, tc.template)
			}
			if !reflect.DeepEqual(c.Args, tc.wantArgs) {
				t.Errorf("%s: Args = %#v; want %#v", tc.name, c.Args, tc.wantArgs)
			}
			if tc.wantParserNil {
				if c.Parser != nil {
					t.Errorf("%s: Parser = %v; want nil", tc.name, c.Parser)
				}
			} else {
				if c.Parser == nil {
					t.Errorf("%s: Parser = nil; want non-nil", tc.name)
				}
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		template string
		args     []any
		want     string
	}{
		{"no_placeholder", "ls -la", nil, "ls -la"},
		{"one_arg", "echo %s", []any{"hello world"}, "echo hello world"},
		{"multiple_verbs", "%d+%d=%d", []any{1, 2, 3}, "1+2=3"},
		{"missing_args", "%s %s", []any{"only"}, "only %!s(MISSING)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &Command{Template: tc.template, Args: tc.args}
			got := cmd.String()
			if got != tc.want {
				t.Errorf("%s: String() = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}
