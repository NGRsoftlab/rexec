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
	defaultMaxSessions = 5
	defaultRetryCount  = 3
	defaultTimeout     = 30 * time.Second
	defaultRetryDelay  = 5 * time.Second
	defaultKeepAlive   = 30 * time.Second
)

type ConfigOption func(*Config) error

// Config contains settings for an SSH connection
type Config struct {
	Host           string        // *ip
	Port           int           // *Port
	User           string        // *username
	timeout        time.Duration // dial timeout
	retryCount     int           // reconnect attempts
	retryInterval  time.Duration // delay between retries
	keepAlive      time.Duration // TCP keepalive interval
	knownHostsPath string        // optional path to known_hosts file to check real SSH Host keys
	// workDir        string // optional, set workdir on Host
	sudoPassword  string            // optional, sudo password
	envVars       map[string]string // remote env vars wrappers
	remoteWorkdir string            // optional workdir
	maxSessions   int               // optional max sessions. if someone want to run commands in parallel

	auth *auth // auth settings
}

// NewConfig creates a new config with defaults and applies any number of configOption.
// Returns an error if any option is invalid or required fields are missing.
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

// WithPort overrides the default SSH Port.
func WithPort(p int) ConfigOption {
	return func(cfg *Config) error {
		if p < 0 || p > 65535 {
			return fmt.Errorf("port must be 1-65535")
		}
		cfg.Port = p
		return nil
	}
}

// WithTimeout sets a custom dial timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(cfg *Config) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be >0")
		}
		cfg.timeout = timeout
		return nil
	}
}

// WithRetry sets how many times to retry dialing and the interval between retries
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

// WithKeepAlive sets a custom TCP keepalive interval
func WithKeepAlive(keepAlive time.Duration) ConfigOption {
	return func(cfg *Config) error {
		if keepAlive <= 0 {
			return fmt.Errorf("keepalive must be >0")
		}
		cfg.keepAlive = keepAlive
		return nil
	}
}

// WithSudoPassword sets the password to use for sudo -S on the remote Host
func WithSudoPassword(password string) ConfigOption {
	// Явный признак того, что нужно ввести sudo пароль, является наличие sudo в шаблоне команды,
	// это нужно отслеживать, мониторить удаленный терминал или еще как-то, когда система запросит
	// пароль от судо и в потоковом режиме вводить пароль от суперпользователя.
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

// WithKnownHosts sets the path to a known_hosts file for server key verification
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

func WithWorkdir(path string) ConfigOption {
	return func(cfg *Config) error {
		if path == "" {
			return fmt.Errorf("workdir path cannot be empty")
		}
		cfg.remoteWorkdir = path
		return nil
	}
}

// WithAgentAuth enables authentication via the local SSH agent
func WithAgentAuth() ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withAgent()
	}
}

// WithKeyBytesAuth enables in-memory private key authentication
func WithKeyBytesAuth(keyBytes []byte, passphrase string) ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withPrivateKeyBytes(keyBytes, passphrase)
	}
}

// WithPrivateKeyPathAuth enables file-based private key authentication
func WithPrivateKeyPathAuth(path, passphrase string) ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withPrivateKeyPath(path, passphrase)
	}
}

// WithPasswordAuth enables password-based authentication
func WithPasswordAuth(password string) ConfigOption {
	return func(cfg *Config) error {
		return cfg.auth.withPassword(password)
	}
}

// validate checks that required fields in Config are set
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

// hostKeyCallback returns a Host key verification function based on knownHostsPath,
// or ssh.InsecureIgnoreHostKey if none specified
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
