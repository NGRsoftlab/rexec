// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngrsoftlab/rexec"
)

type mockWriter struct {
	writes []string
	err    error
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	w.writes = append(w.writes, string(p))
	if w.err != nil {
		return 0, w.err
	}
	return len(p), nil
}

func TestSendSingleFile(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tempFile, []byte("hello scp"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name      string
		content   *rexec.FileContent
		expectErr string
	}{
		{
			"nil content",
			nil,
			"no file content provided",
		},
		{
			"reader",
			&rexec.FileContent{Reader: strings.NewReader("xxx")},
			"",
		},
		{
			"raw data",
			&rexec.FileContent{Data: []byte("abc")},
			"",
		},
		{
			"file source path",
			&rexec.FileContent{SourcePath: tempFile},
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := &mockWriter{}
			w := bufio.NewWriterSize(mw, defaultSCPBufferSize)
			ackBuf := bytes.NewBuffer([]byte{0, 0, 0})
			spec := &rexec.FileSpec{
				TargetDir: filepath.Dir(tempFile),
				Filename:  filepath.Base(tempFile),
				Mode:      0644,
				Content:   tc.content,
			}

			err := sendFile(context.Background(), spec, w, bufio.NewReader(ackBuf))

			if tc.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectErr) {
					t.Errorf("expected error %q, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
