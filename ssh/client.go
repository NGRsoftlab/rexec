// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ngrsoftlab/rexec"
	"github.com/ngrsoftlab/rexec/command"
	"github.com/ngrsoftlab/rexec/parser"
	"github.com/ngrsoftlab/rexec/utils"
	gossh "golang.org/x/crypto/ssh"
)

// interface guard: ensure Client satisfies rexec.Client[RunOption]
var _ rexec.Client[RunOption] = (*Client)(nil)

// Client runs shell commands over an SSH connection
type Client struct {
	cfg    *Config       // SSH connection settings
	client *gossh.Client // active SSH client

	closeOnce      sync.Once             // ensures close actions run only once
	mu             sync.Mutex            // guards client for concurrent use
	keepAliveChan  chan struct{}         // signals keepalive goroutine to stop
	sessionLimiter chan struct{}         // limits concurrent sessions
	mapper         *utils.ExitCodeMapper // maps exit codes to messages
}

// NewClient dials the SSH server using cfg, retrying on failure,
// and starts a keepalive loop. Returns an SSH Client or error
func NewClient(cfg *Config) (*Client, error) {

	sshCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build client config: %w", err)
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	var conn *gossh.Client
	var lastErr error

	for i := 0; i <= cfg.retryCount; i++ {
		conn, lastErr = gossh.Dial("tcp", addr, sshCfg)
		if lastErr == nil {
			break
		}
		time.Sleep(cfg.retryInterval)
	}
	if lastErr != nil {
		return nil, fmt.Errorf("dial failed: %w", lastErr)
	}

	cl := &Client{
		cfg:            cfg,
		client:         conn,
		mapper:         utils.NewDefaultExitCodeMapper(),
		keepAliveChan:  make(chan struct{}),
		sessionLimiter: make(chan struct{}, cfg.maxSessions),
	}

	go cl.keepalive()

	return cl, nil
}

// keepalive periodically sends a no-op request to keep the TCP connection alive
func (cl *Client) keepalive() {
	t := time.NewTicker(cl.cfg.keepAlive)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			cl.mu.Lock()
			_, _, _ = cl.client.Conn.SendRequest("keepalive@openssh.com", false, nil)
			cl.mu.Unlock()
		case <-cl.keepAliveChan:
			return
		}
	}
}

// Session wraps gossh.Session to release a session slot when closed
type Session struct {
	*gossh.Session
	client *Client // parent client to signal limiter
}

// Close closes the SSH session and frees a slot in sessionLimiter
func (w *Session) Close() error {
	err := w.Session.Close()
	<-w.client.sessionLimiter
	return err
}

// OpenSession acquires a session slot, opens a new SSH session, or returns an error
func (cl *Client) OpenSession(ctx context.Context) (*Session, error) {
	select {
	case cl.sessionLimiter <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	cl.mu.Lock()
	sess, err := cl.client.NewSession()
	cl.mu.Unlock()
	if err != nil {
		<-cl.sessionLimiter
		return nil, err
	}

	return &Session{Session: sess, client: cl}, nil
}

// Run executes cmd on the remote host, captures stdout/stderr, exit code, and duration,
// and applies cmd.Parser to dst if provided
func (cl *Client) Run(ctx context.Context, cmd *command.Command, dst any, opts ...RunOption) (*parser.RawResult, error) {
	if cl == nil || cl.client == nil {
		return nil, utils.ErrSessionNotOpen
	}

	result := parser.NewRawResult(cmd)

	var err error
	defer cl.recoverSession(result, &err)

	runCfg := newRunConfig(cl.cfg.remoteWorkdir, cl.cfg.envVars, opts...)
	runCfg.usePTY = cl.requiresPTY(cmd.String())

	sess, err := cl.OpenSession(ctx)
	if err != nil {
		return result, fmt.Errorf("open session: %w", err)
	}
	defer sess.Close()

	if err := cl.requestPTY(sess.Session, runCfg); err != nil {
		return result, err
	}

	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("get stdout pipe: %w", err)
	}
	stderrPipe, err := sess.StderrPipe()
	if err != nil {
		return result, fmt.Errorf("get stderr pipe: %w", err)
	}
	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		return result, fmt.Errorf("get stdin pipe: %w", err)
	}

	cmdStr := cmd.String()
	for k, v := range runCfg.env {
		cmdStr = fmt.Sprintf("export %s=%q; %s", k, v, cmdStr)
	}

	if err := sess.Start(cmdStr); err != nil {
		return result, fmt.Errorf("start command: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		cl.handleStdout(stdoutPipe, stdinPipe, runCfg.stdout)
	}()

	go func() {
		defer wg.Done()
		io.Copy(runCfg.stderr, stderrPipe)
	}()

	if runCfg.stdin != nil {
		go func() {
			io.Copy(stdinPipe, runCfg.stdin)
			stdinPipe.Close()
		}()
	} else {
		defer stdinPipe.Close()
	}

	done := make(chan error, 1)
	go func() {
		done <- sess.Wait()
	}()

	select {
	case <-ctx.Done():
		sess.Close()
		wg.Wait()
		err = ctx.Err()
		result.Err = err
		result.ExitCode = -1
		return result, err

	case e := <-done:
		wg.Wait()
		result.Stdout = runCfg.bufOut.String()
		result.Stderr = runCfg.bufErr.String()

		var exitErr *gossh.ExitError
		if errors.As(e, &exitErr) {
			code := exitErr.ExitStatus()
			msg := cl.mapper.Lookup(code)
			err = fmt.Errorf("remote command failed (%s): %s: %w", msg, result.Stderr, e)
			result.Err = err
			result.ExitCode = code
		} else if e != nil {
			err = e
			result.Err = e
			result.ExitCode = -1
		} else {
			err = nil
			result.ExitCode = 0
		}
	}

	if cmd.Parser != nil && dst != nil {
		if parseErr := cmd.Parser.Parse(result, dst); parseErr != nil {
			result.Err = fmt.Errorf("parse error: %w", parseErr)
		}
	}

	return result, result.Err
}

// Close shuts down keepalive and closes the SSH connection
func (cl *Client) Close() error {
	cl.closeOnce.Do(func() {
		close(cl.keepAliveChan)
	})
	return cl.client.Close()
}

// requiresPTY returns true if shellCmd needs a PTY (e.g., sudo or interactive tools)
func (cl *Client) requiresPTY(shellCmd string) bool {
	keywords := []string{"sudo", "passwd", "su", "ssh", "docker login", "openssl"}
	for _, keyword := range keywords {
		if strings.Contains(shellCmd, keyword) {
			return true
		}
	}
	return false

}

// recoverSession catches panics during Run and records them in result.Err
func (cl *Client) recoverSession(result *parser.RawResult, err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("recovered from panic on run: %v\n%s", r, debug.Stack())
		result.Err = *err
		result.ExitCode = -1
	}
}

// requestPTY asks the server for a pseudo-terminal if runCfg.usePTY is true
func (cl *Client) requestPTY(sess *gossh.Session, runCfg *runConfig) error {
	const (
		term   = "xterm"
		height = 80
		width  = 40
	)

	if !runCfg.usePTY {
		return nil
	}

	modes := gossh.TerminalModes{
		gossh.ECHO:          0,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}

	if err := sess.RequestPty(term, height, width, modes); err != nil {
		return fmt.Errorf("request PTY: %w", err)
	}

	return nil
}

// handleStdout reads lines from stdoutPipe, writes them to stdout writer,
// and automatically responds to password prompts using sudoPassword
func (cl *Client) handleStdout(stdoutPipe io.Reader, stdinPipe io.Writer, stdout io.Writer) {
	passwordPrompt := regexp.MustCompile(`(?i)password\s*:`)
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(stdout, line)
		if passwordPrompt.MatchString(line) && cl.cfg.sudoPassword != "" {
			io.WriteString(stdinPipe, "sudo "+cl.cfg.sudoPassword+"\n")
		}
	}

}
