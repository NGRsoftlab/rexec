package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ngrsoftlab/rexec"
)

type Transfer struct {
}

func NewTransfer() *Transfer {
	return &Transfer{}
}

func (lt *Transfer) Copy(ctx context.Context, spec *rexec.FileSpec) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := lt.validateFileSpec(spec); err != nil {
		return err
	}

	return lt.writeFile(spec)
}

func (lt *Transfer) createDirectory(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	return nil
}

func (lt *Transfer) writeFile(spec *rexec.FileSpec) error {
	fullPath := filepath.Join(spec.TargetDir, spec.Filename)
	parentDir := filepath.Dir(fullPath)

	if err := lt.createDirectory(parentDir, spec.FolderMode); err != nil {
		return err
	}

	reader, _, err := spec.Content.ReaderAndSize()

	outFile, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, spec.Mode)
	if err != nil {
		return fmt.Errorf("create target file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}
	return nil

}

func (lt *Transfer) validateFileSpec(spec *rexec.FileSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	if info, err := os.Stat(spec.TargetDir); err == nil && !info.IsDir() {
		return fmt.Errorf("target dir %q is a file, expected directory", spec.TargetDir)
	}
	return nil
}
