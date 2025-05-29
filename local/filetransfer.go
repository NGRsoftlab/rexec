// Copyright © NGRSoftlab 2020-2025

package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ngrsoftlab/rexec"
)

// Transfer implements FileTransfer by writing files to the local filesystem
type Transfer struct{}

// NewTransfer creates a Transfer for local file operations
func NewTransfer() *Transfer {
	return &Transfer{}
}

// Copy validates spec and writes the file locally.
func (lt *Transfer) Copy(ctx context.Context, spec *rexec.FileSpec) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := lt.validate(spec); err != nil {
		return err
	}

	return lt.writeFile(spec)
}

// createDirectory ensures that the given path exists, creating any necessary parent directories with the specified mode
func (lt *Transfer) createDirectory(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	return nil
}

// writeFile writes spec.Content to TargetDir/Filename, creating parent directories and applying file and folder modes
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

// validate checks that spec is valid and that TargetDir,
// if it exists, is a directory
func (lt *Transfer) validate(spec *rexec.FileSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	if info, err := os.Stat(spec.TargetDir); err == nil && !info.IsDir() {
		return fmt.Errorf("target dir %q is a file, expected directory", spec.TargetDir)
	}
	return nil
}
