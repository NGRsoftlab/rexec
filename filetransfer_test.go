// Copyright © NGRSoftlab 2020-2025

package rexec

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *FileSpec
		wantErr string
	}{
		{name: "nil_spec", spec: nil, wantErr: "file specification empty"},
		{name: "no_filename", spec: &FileSpec{TargetDir: "dir", Content: &FileContent{Data: []byte{1}}}, wantErr: "filename required"},
		{name: "no_target", spec: &FileSpec{Filename: "file", Content: &FileContent{Data: []byte{1}}}, wantErr: "target directory required"},
		{name: "no_content", spec: &FileSpec{Filename: "file", TargetDir: "dir"}, wantErr: "file content required"},
		{name: "empty_content", spec: &FileSpec{Filename: "file", TargetDir: "dir", Content: &FileContent{}}, wantErr: "file content empty"},
		{name: "valid", spec: &FileSpec{Filename: "file", TargetDir: "dir", Content: &FileContent{Data: []byte{1, 2, 3}}}, wantErr: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("%s: unexpected error %v", tc.name, err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("%s: error = %v; want %q", tc.name, err, tc.wantErr)
				}
			}
		})
	}
}

type seekerReader struct{ *bytes.Reader }

type noSeek struct{ io.Reader }

func TestFileContent_ReaderAndSize(t *testing.T) {
	tests := []struct {
		name        string
		typeCase    string
		inputData   []byte
		wantSize    int64
		wantErr     bool
		errContains string
	}{
		{"data", "data", []byte("hello"), 5, false, ""},
		{"source_path", "sourcepath", []byte("abc"), 3, false, ""},
		{"seekable_reader", "readerSeek", []byte("seekable"), 8, false, ""},
		{"nonseekable_reader", "readerNoSeek", []byte("x"), 0, true, "reader is not seekable"},
		{"nil_content", "nil", nil, 0, true, "no file content provided"},
		{"open_error", "statError", nil, 0, true, "open source file"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var content *FileContent
			var expectedData []byte

			switch tc.typeCase {
			case "data":
				content = &FileContent{Data: tc.inputData}
				expectedData = tc.inputData
			case "sourcepath":
				tmpDir := t.TempDir()
				file := filepath.Join(tmpDir, "f.txt")
				if err := os.WriteFile(file, tc.inputData, 0644); err != nil {
					t.Fatalf("setup sourcepath: %v", err)
				}
				content = &FileContent{SourcePath: file}
				expectedData = tc.inputData
			case "readerSeek":
				r := &seekerReader{Reader: bytes.NewReader(tc.inputData)}
				content = &FileContent{Reader: r}
			case "readerNoSeek":
				r := noSeek{Reader: bytes.NewReader(tc.inputData)}
				content = &FileContent{Reader: r}
			case "nil":
				content = nil
			case "statError":
				content = &FileContent{SourcePath: "/nonexistent"}
			}

			r, size, err := content.ReaderAndSize()
			if tc.wantErr {
				if err == nil || (tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains)) {
					t.Errorf("%s: error = %v; want containing %q", tc.name, err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s: unexpected error %v", tc.name, err)
			}
			if size != tc.wantSize {
				t.Errorf("%s: size = %d; want %d", tc.name, size, tc.wantSize)
			}
			if expectedData != nil {
				defer r.Close()
				buf, err := io.ReadAll(r)
				if err != nil {
					t.Fatalf("%s: read error %v", tc.name, err)
				}
				if !bytes.Equal(buf, expectedData) {
					t.Errorf("%s: data = %q; want %q", tc.name, buf, expectedData)
				}
			}
			if r != nil {
				r.Close()
			}
		})
	}
}
