package rexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

type FileTransfer[O any] interface {
	Copy(ctx context.Context, spec *FileSpec, opts ...O) error
}

// type FileTransferOption func(*TransferContext)

type FileContent struct {
	Reader     io.Reader // stream
	Data       []byte    // buffer
	SourcePath string    // filepath
}
type FileSpec struct {
	TargetDir  string // destination path
	Filename   string
	Mode       os.FileMode
	FolderMode os.FileMode // mode for intermediate dirs
	Content    *FileContent
}

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
