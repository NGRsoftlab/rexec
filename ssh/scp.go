// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ngrsoftlab/rexec"
	"github.com/ngrsoftlab/rexec/command"
)

const (
	defaultSCPBufferSize = 2 << 14 // default 32 KB buffer for I/O
	defaultSCPDirMode    = 0o755   // default permission for created directories
)

// SCPOption customizes scpConfig for a transfer
type SCPOption func(config *scpConfig)

// scpConfig holds settings for SCP transfer commands
type scpConfig struct {
	scpBinPath string      // path to the scp executable
	bufSize    int         // size for bufio reader/writer
	folderMode os.FileMode // mode for intermediate directories
}

// newScpConfig creates a config using spec.FolderMode (if >0) and applies opts
func newScpConfig(mode os.FileMode, opts ...SCPOption) *scpConfig {
	cfg := &scpConfig{
		folderMode: defaultSCPDirMode,
		bufSize:    defaultSCPBufferSize,
		scpBinPath: "scp",
	}
	if mode > 0 {
		cfg.folderMode = mode
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithScpBinPath sets a custom scp binary path
func WithScpBinPath(path string) SCPOption {
	return func(config *scpConfig) {
		if path != "" {
			config.scpBinPath = path
		}
	}
}

// WithBufferSize sets a custom bufio buffer size
func WithBufferSize(bufSize int) SCPOption {
	return func(config *scpConfig) {
		if bufSize > 0 {
			config.bufSize = bufSize
		}
	}
}

// SCPTransfer implements FileTransfer by piping data through `scp -t`
type SCPTransfer struct {
	client *Client // underlying SSH client
}

// NewSCPTransfer initializes an SCPTransfer using an SSH client
func NewSCPTransfer(client *Client) *SCPTransfer {
	return &SCPTransfer{client: client}
}

// Copy uploads spec.Content to the remote host via scp.
// It ensures the remote directory exists, starts scp in "to" mode,
// then sends file header, data, and handles acknowledgments
func (t *SCPTransfer) Copy(ctx context.Context, spec *rexec.FileSpec, opts ...SCPOption) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	cfg := newScpConfig(spec.FolderMode, opts...)
	target := escapeShellPath(spec.TargetDir)

	mkdirCmd := command.New(
		"mkdir -p -m %04o %s",
		command.WithArgs(
			spec.FolderMode,
			target,
		),
	)
	if err := rexec.RunNoResult[RunOption](ctx, t.client, mkdirCmd); err != nil {
		return fmt.Errorf("remote mkdir: %w", err)
	}

	sess, err := t.client.OpenSession(ctx)
	if err != nil {
		return fmt.Errorf("open ssh session: %w", err)
	}
	defer sess.Close()

	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		return fmt.Errorf("get stdinPipe pipe: %w", err)
	}
	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("get stdoutPipe pipe: %w", err)
	}

	stderrPipe, err := sess.StderrPipe()
	if err != nil {
		return fmt.Errorf("get stderrPipe pipe: %w", err)
	}

	var errBuf bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- copyWithContext(ctx, stderrPipe, &errBuf)
	}()

	scpCmd := fmt.Sprintf("%s -t %s", cfg.scpBinPath, target)
	if err := sess.Start(scpCmd); err != nil {
		return fmt.Errorf("start scp [%s]: %w -- %s", scpCmd, err, errBuf.String())
	}

	w := bufio.NewWriterSize(stdinPipe, cfg.bufSize)
	r := bufio.NewReaderSize(stdoutPipe, cfg.bufSize)

	// init ACK
	if err := readAck(ctx, r); err != nil {
		return fmt.Errorf("initial ACK: %w", err)
	}

	if err := sendFile(ctx, spec, w, r); err != nil {
		return fmt.Errorf("send file %q: %w", spec.Filename, err)
	}

	if err := stdinPipe.Close(); err != nil {
		return fmt.Errorf("close stdinPipe: %w", err)
	}

	if waitErr := sess.Wait(); waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			code := exitErr.ExitCode()
			msg := t.client.mapper.Lookup(code)
			return fmt.Errorf("scp failed (%s): %w", msg, waitErr)
		}
		if drainErr := <-errCh; drainErr != nil {
			return fmt.Errorf("scp failed: %w -- %s", drainErr, errBuf.String())
		}
		return fmt.Errorf("scp failed: %w -- %s", waitErr, errBuf.String())
	}
	<-errCh
	return nil
}

// sendFile follows SCP protocol: header → ACK → data → EOF byte → ACK
func sendFile(ctx context.Context, spec *rexec.FileSpec, w *bufio.Writer, r *bufio.Reader) error {
	reader, size, err := spec.Content.ReaderAndSize()
	if err != nil {
		return err
	}
	defer reader.Close()

	header := fmt.Sprintf("C%04o %d %s\n", spec.Mode.Perm(), size, spec.Filename)
	if _, err := w.WriteString(header); err != nil {
		return fmt.Errorf("write file header: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush file header: %w", err)
	}
	if err := readAck(ctx, r); err != nil {
		return fmt.Errorf("ACK after header: %w", err)
	}

	// data
	if err := copyWithContext(ctx, reader, w); err != nil {
		return fmt.Errorf("send file data: %w", err)
	}

	// EOF-byte
	if err := w.WriteByte(0); err != nil {
		return fmt.Errorf("write EOF byte: %w", err)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush file data: %w", err)
	}

	if err := readAck(ctx, r); err != nil {
		return fmt.Errorf("final ACK: %w", err)
	}
	return nil
}

// readAck reads one status byte and returns error if non-zero
func readAck(ctx context.Context, r *bufio.Reader) error {
	if err := ctx.Err(); err != nil {
		return ctx.Err()
	}
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("read ack: %w", err)
	}
	if b != 0 {
		msg, _ := r.ReadString('\n')
		return fmt.Errorf("scp error: %s", strings.TrimSpace(msg))
	}
	return nil
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, defaultSCPBufferSize)
	},
}

// copyWithContext copies from src to dst in chunks, aborting on context cancel
func copyWithContext(ctx context.Context, src io.Reader, dst io.Writer) error {
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return nil
			}
			return rerr
		}
	}
}

// escapeShellPath safely quotes a path for sh single-quoted strings
func escapeShellPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
