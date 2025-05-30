// Copyright © NGRSoftlab 2020-2025

package examples

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ngrsoftlab/rexec/parser"
)

func TestLsEntry_ParsePermissions(t *testing.T) {
	tests := []struct {
		name     string
		permStr  string
		wantMode uint32
		wantErr  bool
	}{
		{name: "dir_rw", permStr: "drw-r--r--", wantMode: uint32(0400|0200|0040|0004) | uint32(os.ModeDir), wantErr: false},
		{name: "symlink_x", permStr: "lrwxrwxrwx", wantMode: uint32(0400|0200|0100|0040|0020|0010|0004|0002|0001) | uint32(os.ModeSymlink), wantErr: false},
		{name: "char_device", permStr: "crw-------", wantMode: uint32(0400|0200) | uint32(os.ModeDevice), wantErr: false},
		{name: "pipe", permStr: "prw-rw----", wantMode: uint32(0400|0200|0040|0020) | uint32(os.ModeNamedPipe), wantErr: false},
		{name: "bad_format", permStr: "invalid", wantMode: 0, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &LsEntry{Permissions: tc.permStr}
			mode, err := e.ParsePermissions()
			if (err != nil) != tc.wantErr {
				t.Fatalf("%s: error = %v; wantErr %v", tc.name, err, tc.wantErr)
			}
			if err == nil && uint32(mode) != tc.wantMode {
				t.Errorf("%s: mode = %o; want %o", tc.name, mode, tc.wantMode)
			}
		})
	}
}

func TestLsParser_Parse(t *testing.T) {
	raw := &parser.RawResult{Stdout: `total 2
-rw-r--r-- 1 user group 123 Jan  1 12:00 file1
lrwxrwxrwx 2 alice staff  64 Feb 28 2021 link -> target
invalid line
-rw------- 3 bob  dev   456 Mar 10 15:30 spaced file name.txt
`}

	t.Run("wrong_dst", func(t *testing.T) {
		var dst map[string]int
		err := (&LsParser{}).Parse(raw, &dst)
		if err == nil || err.Error() != "dst must be *[]LsEntry" {
			t.Errorf("wrong_dst: err = %v; want dst must be *[]LsEntry", err)
		}
	})

	t.Run("invalid_links", func(t *testing.T) {
		bad := &parser.RawResult{Stdout: "total 1\n-rw-r--r-- X user group 123 Jan 1 00:00 f"}
		var out []LsEntry
		err := (&LsParser{}).Parse(bad, &out)
		if err == nil || !strings.Contains(err.Error(), "invalid links") {
			t.Errorf("invalid_links: err = %v; want invalid links", err)
		}
	})

	t.Run("invalid_size", func(t *testing.T) {
		bad := &parser.RawResult{Stdout: "total 1\n-rw-r--r-- 1 user group XYZ Jan 1 00:00 f"}
		var out []LsEntry
		err := (&LsParser{}).Parse(bad, &out)
		if err == nil || !strings.Contains(err.Error(), "invalid size") {
			t.Errorf("invalid_size: err = %v; want invalid size", err)
		}
	})

	t.Run("parse_entries", func(t *testing.T) {
		var got []LsEntry
		err := (&LsParser{}).Parse(raw, &got)
		if err != nil {
			t.Fatalf("parse_entries: unexpected error %v", err)
		}
		want := []LsEntry{
			{Permissions: "-rw-r--r--", Links: 1, Owner: "user", Group: "group", Size: 123, Month: "Jan", Day: "1", TimeOrYear: "12:00", Name: "file1"},
			{Permissions: "lrwxrwxrwx", Links: 2, Owner: "alice", Group: "staff", Size: 64, Month: "Feb", Day: "28", TimeOrYear: "2021", Name: "link -> target"},
			{Permissions: "-rw-------", Links: 3, Owner: "bob", Group: "dev", Size: 456, Month: "Mar", Day: "10", TimeOrYear: "15:30", Name: "spaced file name.txt"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("parse_entries: got %+v; want %+v", got, want)
		}
	})
}
