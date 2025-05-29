package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ngrsoftlab/rexec"
	"github.com/pkg/sftp"
)

const (
	defaultSFTPBufferSize = 2 << 14 // default 32 KB transfer buffer
	defaultSFTPDirMode    = 0o755   // default mode for created directories
)

// SFTPOption customizes SFTP transfer behavior
type SFTPOption func(*sftpConfig)

type sftpConfig struct {
	bufferSize int         // size for copy buffer
	folderMode os.FileMode // mode for directories
}

// newSFTPConfig builds a config using spec.FolderMode (if non-zero) and opts
func newSFTPConfig(mode os.FileMode, opts ...SFTPOption) *sftpConfig {
	cfg := &sftpConfig{
		bufferSize: defaultSFTPBufferSize,
		folderMode: defaultSFTPDirMode,
	}

	if mode != 0 {
		cfg.folderMode = mode
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithSFTPBufferSize sets the buffer size for io.Copy
func WithSFTPBufferSize(n int) SFTPOption {
	return func(c *sftpConfig) {
		if n > 0 {
			c.bufferSize = n
		}
	}
}

// SFTPTransfer implements FileTransfer over SSH using the SFTP subsystem
type SFTPTransfer struct {
	client *Client
}

// NewSFTPTransfer creates an SFTPTransfer tied to the given SSH client
func NewSFTPTransfer(client *Client) *SFTPTransfer {
	return &SFTPTransfer{client: client}
}

// Copy uploads spec.Content to spec.TargetDir on the remote host via SFTP.
// It creates directories, writes the file, and applies permissions
func (t *SFTPTransfer) Copy(ctx context.Context, spec *rexec.FileSpec, opts ...SFTPOption) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	cfg := newSFTPConfig(spec.FolderMode, opts...)

	sftpCli, sess, err := t.openSFTPSession(ctx)
	if err != nil {
		return err
	}
	defer func() {
		sftpCli.Close()
		sess.Close()
		sess.Wait()
	}()

	if err := sftpCli.MkdirAll(spec.TargetDir); err != nil {
		return fmt.Errorf("sftp create target dir: %w", err)
	}
	if err := sftpCli.Chmod(spec.TargetDir, cfg.folderMode); err != nil {
		return fmt.Errorf("sftp chmod dir: %w", err)
	}

	remotePath := path.Join(spec.TargetDir, spec.Filename)
	f, err := sftpCli.OpenFile(remotePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp open file: %w", err)
	}
	defer f.Close()

	reader, _, err := spec.Content.ReaderAndSize()
	if err != nil {
		return fmt.Errorf("sftp read source data: %w", err)
	}
	defer reader.Close()

	buf := make([]byte, cfg.bufferSize)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rErr := reader.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return fmt.Errorf("sftp write remote data: %w", err)
			}
		}
		if rErr != nil {
			if errors.Is(rErr, io.EOF) {
				break
			}
			return fmt.Errorf("sftp read source data: %w", rErr)
		}
	}

	if err := f.Chmod(spec.Mode); err != nil {
		return fmt.Errorf("sftp chmod file: %w", err)
	}
	return nil

}

// openSFTPSession starts an SSH session, requests the "sftp" subsystem,
// and returns a sftp.Client over the session pipes
func (t *SFTPTransfer) openSFTPSession(ctx context.Context) (*sftp.Client, *Session, error) {
	sess, err := t.client.OpenSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("open ssh session for sftp: %w", err)
	}

	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		sess.Close()
		return nil, nil, fmt.Errorf("get sftp stdout pipe: %w", err)
	}
	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		return nil, nil, fmt.Errorf("get sftp stdin pipe: %w", err)
	}

	if err := sess.RequestSubsystem("sftp"); err != nil {
		sess.Close()
		return nil, nil, fmt.Errorf("request sftp subsystem: %w", err)
	}

	cli, err := sftp.NewClientPipe(stdoutPipe, stdinPipe)
	if err != nil {
		sess.Close()
		return nil, nil, fmt.Errorf("sftp new client pipe: %w", err)
	}
	return cli, sess, nil
}
