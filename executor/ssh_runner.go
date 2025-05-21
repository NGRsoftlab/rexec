package executor

import (
	"sync"

	"github.com/ngrsoftlab/rexec/config"
	"golang.org/x/crypto/ssh"
)

// SSHRunner implements CommandRunner for remote SSH command execution.
type SSHRunner struct {
	client   *ssh.Client
	config   *config.Config
	mu       sync.Mutex
	isClosed bool
}

// New establishes a new SSH connection using the given config.
// func New(cfg *config.Config) (*SSHRunner, error) {
// 	if cfg == nil {
// 		return nil, fmt.Errorf("config cannot be nil")
// 	}
//
// 	sshConfig := &ssh.ClientConfig{
// 		User:            cfg.User,
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: make this configurable
// 		Timeout:         cfg.Timeout,
// 	}
//
// 	switch cfg.Auth.Type {
// 	case config.SSHAuthPassword:
// 		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(cfg.Auth.Password)}
// 	case config.SSHAuthPrivateKeyPath:
// 		sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(cfg.Auth.Signer)}
// 	default:
// 		return nil, fmt.Errorf("unsupported SSH auth type")
// 	}
//
// 	conn, err := ssh.Dial("tcp", cfg.Host, sshConfig)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to dial SSH: %w", err)
// 	}
//
// 	return &SSHRunner{
// 		client:   conn,
// 		config:   cfg,
// 		isClosed: false,
// 	}, nil
// }
//
// func (r *SSHRunner) RunCommand(ctx context.Context, cmd string) *CommandResult {
// 	result := &CommandResult{}
//
// 	r.mu.Lock()
// 	if r.isClosed {
// 		r.mu.Unlock()
// 		result.Err = fmt.Errorf("executor is closed")
// 		return result
// 	}
// 	r.mu.Unlock()
//
// 	session, err := r.client.NewSession()
// 	if err != nil {
// 		result.Err = fmt.Errorf("failed to create session: %w", err)
// 		return result
// 	}
//
// 	defer session.Close()
//
// 	var stdoutBuf, stderrBuf bytes.Buffer
// 	session.Stdout = &stdoutBuf
// 	session.Stderr = &stderrBuf
//
// 	finalCmd := cmd
// 	if r.config.SudoPassword != "" && requiresSudo(cmd) {
// 		finalCmd = fmt.Sprintf("echo %q | sudo -S %s", r.config.SudoPassword, cmd)
// 	}
//
// 	errCh := make(chan error, 1)
// 	go func() {
// 		errCh <- session.Run(finalCmd)
// 	}()
//
// 	select {
// 	case <-ctx.Done():
// 		switch ctx.Err() {
// 		case context.DeadlineExceeded:
//
// 		}
// 		_ = session.Signal(ssh.SIGKILL)
// 		result.Err = fmt.Errorf("command timed out or canceled: %w", ctx.Err())
// 		result.ExitCode = -1
// 	case err := <-errCh:
// 		if err != nil {
// 			var exitErr *ssh.ExitError
// 			if errors.As(err, &exitErr) {
// 				result.ExitCode = exitErr.ExitStatus()
// 			} else {
// 				result.Err = fmt.Errorf("command failed: %w", err)
// 				result.ExitCode = -1
// 			}
// 		}
// 	}
//
// 	result.Stdout = stdoutBuf.Bytes()
// 	result.Stderr = stderrBuf.Bytes()
//
// 	return result
// }
//
// func (r *SSHRunner) CopyFile(ctx context.Context, src io.Reader, destPath string) error {
// 	r.mu.Lock()
// 	if r.isClosed {
// 		r.mu.Unlock()
// 		return fmt.Errorf("executor is closed")
// 	}
// 	r.mu.Unlock()
//
// 	session, err := r.client.NewSession()
// 	if err != nil {
// 		return fmt.Errorf("failed to create session: %w", err)
// 	}
// 	defer session.Close()
//
// 	stdin, err := session.StdinPipe()
// 	if err != nil {
// 		return fmt.Errorf("failed to create stdin pipe: %w", err)
// 	}
// 	stdout, err := session.StdoutPipe()
// 	if err != nil {
// 		return fmt.Errorf("failed to create stdout pipe: %w", err)
// 	}
//
// 	if err := session.Start(fmt.Sprintf("scp -qt %s", shellEscape(filepath.Dir(destPath)))); err != nil {
// 		return fmt.Errorf("failed to start remote scp: %w", err)
// 	}
//
// 	buf := new(bytes.Buffer)
// 	n, err := io.Copy(buf, src)
// 	if err != nil {
// 		return fmt.Errorf("failed to read file data: %w", err)
// 	}
//
// 	filename := filepath.Base(destPath)
// 	mode := "0644"
//
// 	writer := bufio.NewWriter(stdin)
// 	fmt.Fprintf(writer, "C%s %d %s\n", mode, n, filename)
// 	writer.Flush()
// }
//
// // requiresSudo determines if the command starts with 'sudo'.
// func requiresSudo(cmd string) bool {
// 	return len(cmd) >= 4 && cmd[:4] == "sudo"
// }
//
// // Close shuts down the SSH connection.
// func (r *SSHRunner) Close() error {
// 	if r.isClosed {
// 		return nil
// 	}
// 	r.isClosed = true
// 	return r.client.Close()
// }
