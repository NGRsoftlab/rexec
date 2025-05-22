package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHOption func(*SSHConfig) error

var (
	DefaultPort       = 22
	DefaultTimeout    = 30 * time.Second
	DefaultRetryCount = 3
	DefaultRetryDelay = 5 * time.Second
	DefaultKeepAlive  = 30 * time.Second
)

// SSHConfig contains settings for an SSH connection
type SSHConfig struct {
	Host           string        // *ip
	Port           int           // *port
	User           string        // *username
	Timeout        time.Duration // dial timeout
	KeepAlive      time.Duration // TCP keepalive interval
	RetryCount     int           // reconnect attempts
	RetryInterval  time.Duration // delay between retries
	KnownHostsPath string        // optional path to known_hosts file to check real SSH host keys
	// WorkDir        string // optional, set workdir on host
	SudoPassword string            // optional, sudo password
	EnvVars      map[string]string // remote env vars wrappers

	Auth *SSHAuth // auth settings
}

// NewSSHConfig creates a new SSHConfig with defaults and applies any number of SSHOption.
// Returns an error if any option is invalid or required fields are missing.
func NewSSHConfig(user, host string, port int, opts ...SSHOption) (*SSHConfig, error) {
	cfg := &SSHConfig{
		Host:          host,
		Port:          port,
		User:          user,
		Timeout:       DefaultTimeout,
		RetryCount:    DefaultRetryCount,
		RetryInterval: DefaultRetryDelay,
		KeepAlive:     DefaultKeepAlive,
		EnvVars:       make(map[string]string),
		Auth:          &SSHAuth{},
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("ssh config option failed: %w", err)
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("ssh config validation failed: %w", err)
	}

	return cfg, nil
}

// ===== SSHOptions =====

// WithPort overrides the default SSH port.
func WithPort(p int) SSHOption {
	return func(cfg *SSHConfig) error {
		if p < 0 || p > 65535 {
			return fmt.Errorf("ssh: port must be 1-65535")
		}
		cfg.Port = p
		return nil
	}
}

// WithTimeout sets a custom dial timeout
func WithTimeout(timeout time.Duration) SSHOption {
	return func(cfg *SSHConfig) error {
		if timeout <= 0 {
			return fmt.Errorf("ssh: timeout must be >0")
		}
		cfg.Timeout = timeout
		return nil
	}
}

// WithRetry sets how many times to retry dialing and the interval between retries
func WithRetry(count int, interval time.Duration) SSHOption {
	return func(cfg *SSHConfig) error {
		if count < 0 || interval < 0 {
			return fmt.Errorf("ssh: retry count or interval must be >0")
		}
		cfg.RetryCount = count
		cfg.RetryInterval = interval
		return nil
	}
}

// WithKeepAlive sets a custom TCP keepalive interval
func WithKeepAlive(keepAlive time.Duration) SSHOption {
	return func(cfg *SSHConfig) error {
		if keepAlive <= 0 {
			return fmt.Errorf("ssh: keepalive must be >0")
		}
		cfg.KeepAlive = keepAlive
		return nil
	}
}

// WithSudoPassword sets the password to use for sudo -S on the remote host
func WithSudoPassword(password string) SSHOption {
	return func(cfg *SSHConfig) error {
		cfg.SudoPassword = password
		return nil
	}
}

// WithEnvVars merges provided environment variables into the remote session
func WithEnvVars(envVars map[string]string) SSHOption {
	return func(cfg *SSHConfig) error {
		for k, v := range envVars {
			cfg.EnvVars[k] = v
		}
		return nil
	}
}

// WithKnownHosts sets the path to a known_hosts file for server key verification
func WithKnownHosts(path string) SSHOption {
	return func(cfg *SSHConfig) error {
		if path == "" {
			return fmt.Errorf("ssh: known_hosts path cannot be empty")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("ssh: known_hosts file '%s' does not exist", filepath.Base(path))
		}
		cfg.KnownHostsPath = path
		return nil
	}
}

// WithAgentAuth enables authentication via the local SSH agent
func WithAgentAuth() SSHOption {
	return func(cfg *SSHConfig) error {
		return cfg.Auth.withAgent()
	}
}

// WithKeyBytesAuth enables in-memory private key authentication
func WithKeyBytesAuth(keyBytes []byte, passphrase string) SSHOption {
	return func(cfg *SSHConfig) error {
		return cfg.Auth.withPrivateKeyBytes(keyBytes, passphrase)
	}
}

// WithPrivateKeyPathAuth enables file-based private key authentication
func WithPrivateKeyPathAuth(path, passphrase string) SSHOption {
	return func(cfg *SSHConfig) error {
		return cfg.Auth.withPrivateKeyPath(path, passphrase)
	}
}

// WithPasswordAuth enables password-based authentication
func WithPasswordAuth(password string) SSHOption {
	return func(cfg *SSHConfig) error {
		return cfg.Auth.withPassword(password)
	}
}

// Validate checks that required fields in SSHConfig are set
func (c *SSHConfig) Validate() error {
	if len(c.User) == 0 {
		return fmt.Errorf("user required")
	}
	if len(c.Host) == 0 {
		return fmt.Errorf("host required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// ClientConfig builds the underlying *ssh.ClientConfig, gathering auth methods
// in priority order (agent → keyPath/bytes → password) and setting the host-key callback
func (c *SSHConfig) ClientConfig() (*ssh.ClientConfig, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("ssh: invalid config: %w", err)
	}

	authMethods, err := c.Auth.authMethods()
	if err != nil {
		return nil, fmt.Errorf("ssh: invalid auth methods: %w", err)
	}

	hostKeyCallback, err := c.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            c.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.Timeout,
	}

	return clientConfig, nil
}

// hostKeyCallback returns a host key verification function based on KnownHostsPath,
// or ssh.InsecureIgnoreHostKey if none specified
func (c *SSHConfig) hostKeyCallback() (ssh.HostKeyCallback, error) {
	hostCallback := ssh.InsecureIgnoreHostKey()
	if len(c.KnownHostsPath) > 0 {
		callback, err := knownhosts.New(c.KnownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("ssh: knownhost: %w", err)
		}
		hostCallback = callback
	}
	return hostCallback, nil
}
