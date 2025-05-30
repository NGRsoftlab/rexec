// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	defaultMaxSessions = 1                // default maximum concurrent SSH sessions
	defaultRetryCount  = 3                // default number of connection retries
	defaultTimeout     = 30 * time.Second // default dial timeout
	defaultRetryDelay  = 5 * time.Second  // default delay between retry attempts
	defaultKeepAlive   = 30 * time.Second // default TCP keepalive interval
)

// ConfigOption customizes SSH Config settings
type ConfigOption func(*Config) error

// Config holds settings for establishing and managing an SSH connection
type Config struct {
	Host           string            // *remote host IP or hostname
	Port           int               // *SSH port number
	User           string            // *SSH username
	timeout        time.Duration     //  dial timeout duration
	retryCount     int               // reconnect attempts
	retryInterval  time.Duration     // delay between retries
	keepAlive      time.Duration     // TCP keepalive interval
	knownHostsPath string            // path to known_hosts for host key verification
	sudoPassword   string            // optional: password for sudo operations on remote host
	envVars        map[string]string // environment variables to set on remote session
	remoteWorkdir  string            // optional: working directory on the remote host
	maxSessions    int               // optional: max concurrent sessions per connection

	auth *auth // authentication settings
}

// NewConfig creates a Config with required user, host, port and applies any options.
// Returns an error if any option fails or required fields are invalid
func NewConfig(user, host string, port int, opts ...ConfigOption) (*Config, error) {
	cfg := &Config{
		Host:          host,
		Port:          port,
		User:          user,
		timeout:       defaultTimeout,
		retryCount:    defaultRetryCount,
		retryInterval: defaultRetryDelay,
		keepAlive:     defaultKeepAlive,
		envVars:       make(map[string]string),
		auth:          &auth{},
		maxSessions:   defaultMaxSessions,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("config option failed: %w", err)
		}
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// ===== SSHOptions =====

// WithPort overrides the SSH port
func WithPort(p int) ConfigOption {
	return func(cfg *Config) error {
		if p <= 0 || p > 65535 {
			return fmt.Errorf("port must be 1-65535")
		}
		cfg.Port = p
		return nil
	}
}

// WithTimeout sets the dial timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(cfg *Config) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be >0")
		}
		cfg.timeout = timeout
		return nil
	}
}

// WithRetry sets connection retry count and interval
func WithRetry(count int, interval time.Duration) ConfigOption {
	return func(cfg *Config) error {
		if count < 0 || interval < 0 {
			return fmt.Errorf("retry count or interval must be >0")
		}
		cfg.retryCount = count
		cfg.retryInterval = interval
		return nil
	}
}

// WithKeepAlive sets the TCP keepalive interval
func WithKeepAlive(keepAlive time.Duration) ConfigOption {
	return func(cfg *Config) error {
		if keepAlive <= 0 {
			return fmt.Errorf("keepalive must be >0")
		}
		cfg.keepAlive = keepAlive
		return nil
	}
}

// WithSudoPassword configures a password for sudo -S on the remote host
func WithSudoPassword(password string) ConfigOption {
	return func(cfg *Config) error {
		if password == "" {
			return fmt.Errorf("password must not be empty")
		}
		cfg.sudoPassword = password
		return nil
	}
}

// WithEnvVars merges provided environment variables into the remote session
func WithEnvVars(envVars map[string]string) ConfigOption {
	return func(cfg *Config) error {
		for k, v := range envVars {
			cfg.envVars[k] = v
		}
		return nil
	}
}

// WithKnownHosts sets the path to a known_hosts file for host key checking
func WithKnownHosts(path string) ConfigOption {
	return func(cfg *Config) error {
		if path == "" {
			return fmt.Errorf("known_hosts path cannot be empty")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("known_hosts file '%s' does not exist", filepath.Base(path))
		}
		cfg.knownHostsPath = path
		return nil
	}
}

// WithWorkdir sets the remote working directory
func WithWorkdir(path string) ConfigOption {
	return func(cfg *Config) error {
		if path == "" {
			return fmt.Errorf("workdir path cannot be empty")
		}
		cfg.remoteWorkdir = path
		return nil
	}
}

// WithMaxSessions - set max concurrent sessions for connection. You can see it on host in /etc/ssh/sshd_config.
// Recommend value between 1 and 4
func WithMaxSessions(maxSessions int) ConfigOption {
	return func(cfg *Config) error {
		if maxSessions <= 0 || maxSessions > 6 {
			return fmt.Errorf("max sessions must be between 1 and 6")
		}
		cfg.maxSessions = maxSessions
		return nil
	}
}

// WithAgentAuth enables SSH agent-based authentication
func WithAgentAuth() ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withAgent()
	}
}

// WithKeyBytesAuth enables private key authentication using in-memory key bytes
func WithKeyBytesAuth(keyBytes []byte, passphrase string) ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withPrivateKeyBytes(keyBytes, passphrase)
	}
}

// WithPrivateKeyPathAuth enables private key authentication from a file
func WithPrivateKeyPathAuth(path, passphrase string) ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withPrivateKeyPath(path, passphrase)
	}
}

// WithPasswordAuth enables password-based SSH authentication
func WithPasswordAuth(password string) ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withPassword(password)
	}
}

// validate ensures required Config fields are set correctly
func (c *Config) validate() error {
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
// in priority order (agent → keyPath/bytes → password) and setting the Host-key callback
func (c *Config) ClientConfig() (*ssh.ClientConfig, error) {
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	authMethods, err := c.auth.authMethods()
	if err != nil {
		return nil, fmt.Errorf("invalid auth methods: %w", err)
	}

	hostKeyCallback, err := c.hostKeyCallback()
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            c.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.timeout,
	}

	return clientConfig, nil
}

// hostKeyCallback returns a HostKeyCallback based on knownHostsPath,
// or ssh.InsecureIgnoreHostKey if none is specified
func (c *Config) hostKeyCallback() (ssh.HostKeyCallback, error) {
	hostCallback := ssh.InsecureIgnoreHostKey()
	if len(c.knownHostsPath) > 0 {
		callback, err := knownhosts.New(c.knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("knownhost: %w", err)
		}
		hostCallback = callback
	}
	return hostCallback, nil
}
