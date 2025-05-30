// Copyright © NGRSoftlab 2020-2025

package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ngrsoftlab/rexec"
)

func TestTransfer_Copy(t *testing.T) {
	tmpDir := t.TempDir()
	validData := []byte("hello")

	tests := []struct {
		name      string
		spec      *rexec.FileSpec
		ctx       context.Context
		setupErr  bool
		wantErr   string
		checkFile bool
		wantData  []byte
	}{
		{
			name:     "nil_spec",
			spec:     nil,
			ctx:      context.Background(),
			setupErr: true,
			wantErr:  "file specification empty",
		},
		{
			name:     "validate_fail",
			spec:     &rexec.FileSpec{TargetDir: tmpDir, Filename: "", Content: &rexec.FileContent{Data: validData}},
			ctx:      context.Background(),
			setupErr: true,
			wantErr:  "filename required",
		},
		{
			name:    "ctx_canceled",
			spec:    &rexec.FileSpec{TargetDir: tmpDir, Filename: "f.txt", Content: &rexec.FileContent{Data: validData}},
			ctx:     func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }(),
			wantErr: context.Canceled.Error(),
		},
		{
			name:      "success",
			spec:      &rexec.FileSpec{TargetDir: tmpDir, Filename: "out.txt", Mode: 0644, FolderMode: 0755, Content: &rexec.FileContent{Data: validData}},
			ctx:       context.Background(),
			wantErr:   "",
			checkFile: true,
			wantData:  validData,
		},
	}

	tr := NewTransfer()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tr.Copy(tc.ctx, tc.spec)
			if tc.wantErr != "" {
				if err == nil || !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr)) {
					t.Fatalf("%s: err = %v; want containing %q", tc.name, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s: unexpected error %v", tc.name, err)
			}
			if tc.checkFile {
				path := filepath.Join(tc.spec.TargetDir, tc.spec.Filename)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("%s: read file error %v", tc.name, err)
				}
				if !bytes.Equal(data, tc.wantData) {
					t.Errorf("%s: data = %q; want %q", tc.name, data, tc.wantData)
				}
			}
		})
	}
}
