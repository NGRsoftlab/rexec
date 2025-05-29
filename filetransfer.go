// Copyright © NGRSoftlab 2020-2025

package rexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

// FileTransfer defines an interface for copying files according to a FileSpec
type FileTransfer[O any] interface {
	// Copy transfers the file described by spec, applying any transfer options
	Copy(ctx context.Context, spec *FileSpec, opts ...O) error
}

// FileContent holds the source of file data for transfer.
// Only one of Data, SourcePath, or Reader should be set
type FileContent struct {
	Reader     io.Reader // stream to read file data from
	Data       []byte    // in-memory file data
	SourcePath string    // path to the file on disk
}

// FileSpec describes where and how to create a file on the target
type FileSpec struct {
	TargetDir  string       // destination directory
	Filename   string       // name of the file to create
	Mode       os.FileMode  // file permission bits
	FolderMode os.FileMode  // permission bits for any created directories
	Content    *FileContent // file data and source information
}

// Validate checks that the spec has all required fields and content
func (t *FileSpec) Validate() error {
	if t == nil {
		return fmt.Errorf("file specification empty")
	}
	if t.Filename == "" {
		return fmt.Errorf("filename required")
	}
	if t.TargetDir == "" {
		return fmt.Errorf("target directory required")
	}
	if t.Content == nil {
		return fmt.Errorf("file content required")
	}
	if len(t.Content.Data) == 0 && t.Content.SourcePath == "" && t.Content.Reader == nil {
		return fmt.Errorf("file content empty")
	}
	return nil
}

// ReaderAndSize yields an io.ReadCloser and its length based on which
// content field is set: Data, SourcePath, or Reader
func (t *FileContent) ReaderAndSize() (io.ReadCloser, int64, error) {
	switch {
	case t == nil:
		return nil, 0, fmt.Errorf("no file content provided")
	case len(t.Data) > 0:
		return io.NopCloser(bytes.NewReader(t.Data)), int64(len(t.Data)), nil
	case t.SourcePath != "":
		f, err := os.Open(t.SourcePath)
		if err != nil {
			return nil, 0, fmt.Errorf("open source file: %w", err)
		}
		info, err := f.Stat()
		if err != nil {
			if err := f.Close(); err != nil {
				return nil, 0, fmt.Errorf("close source file: %w", err)
			}
			return nil, 0, fmt.Errorf("stat source file: %w", err)
		}
		return f, info.Size(), nil
	case t.Reader != nil:
		if s, ok := t.Reader.(io.Seeker); ok {
			cur, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, 0, fmt.Errorf("seek curent source file: %w", err)
			}
			end, err := s.Seek(0, io.SeekEnd)
			if err != nil {
				return nil, 0, fmt.Errorf("seek end source file: %w", err)
			}
			_, err = s.Seek(cur, io.SeekStart)
			return io.NopCloser(t.Reader), end - cur, err
		} else {
			return nil, 0, fmt.Errorf("reader is not seekable")
		}
	default:
		return nil, 0, fmt.Errorf("no file content provided")
	}
}
