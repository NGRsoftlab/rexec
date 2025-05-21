package config

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	DefaultTimeout    = 30 * time.Second
	DefaultRetryCount = 3
	DefaultRetryDelay = 5 * time.Second
	DefaultKeepAlive  = 30 * time.Second
)

// Config contains settings for an SSH connection
type Config struct {
	User          string            // *username
	Host          string            // *ip
	Port          uint16            // *port
	Auth          *Auth             // auth settings
	Timeout       time.Duration     // dial timeout
	KeepAlive     time.Duration     // TCP keepalive interval
	RetryCount    int               // reconnect attempts
	RetryInterval time.Duration     // delay between retries
	SudoPassword  string            // optional, if `sudo -S` is required
	EnvVars       map[string]string // TODO: maybe move to session. environment variables to set
}

// NewConfig - create new config for ssh connection
func NewConfig(user, host string, port uint16) *Config {
	cfg := &Config{
		User:          user,
		Host:          host,
		Port:          port,
		Timeout:       DefaultTimeout,
		KeepAlive:     DefaultKeepAlive,
		RetryCount:    DefaultRetryCount,
		RetryInterval: DefaultRetryDelay,
		EnvVars:       make(map[string]string),
	}
	return cfg
}

// WithPasswordAuth sets password authentication
func (c *Config) WithPasswordAuth(password string) *Config {
	if c.Auth == nil {
		c.Auth = &Auth{}
	}
	c.Auth.withPassword(password)
	return c
}

// WithPrivateKeyPathAuth sets private key file authentication. Passphrase can be empty
func (c *Config) WithPrivateKeyPathAuth(path, passphrase string) *Config {
	if c.Auth == nil {
		c.Auth = &Auth{}
	}
	c.Auth.withPrivateKeyPath(path, passphrase)
	return c
}

// WithPrivateKeyBytesAuth sets private key bytes authentication. Passphrase can be empty
func (c *Config) WithPrivateKeyBytesAuth(key []byte, passphrase string) *Config {
	if c.Auth == nil {
		c.Auth = &Auth{}
	}
	c.Auth.withPrivateKeyBytes(key, passphrase)
	return c
}

// WithAgentAuth enables SSH agent authentication
func (c *Config) WithAgentAuth() *Config {
	if c.Auth == nil {
		c.Auth = &Auth{}
	}
	c.Auth.withAgent()
	return c
}

// WithSudoPassword - sets sudo password for remote elevated commands
func (c *Config) WithSudoPassword(password string) *Config {
	if len(password) != 0 {
		c.SudoPassword = password
	}
	return c
}

// WithEnvVars - merge environment variables for remote commands
func (c *Config) WithEnvVars(envVars map[string]string) *Config {
	for k, v := range envVars {
		c.EnvVars[k] = v
	}
	return c
}

// Validate ensures the config has all required fields
func (c *Config) Validate() error {
	if c.Host == "" {
		return errors.New("host must be provided")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if c.User == "" {
		return errors.New("user must be provided")
	}
	if len(c.Auth.Methods) == 0 {
		return errors.New("at least one auth method must be provided")
	}

	return nil
}

// ClientConfig - builds an *ssh.ClientConfig for use in dialing
func (c *Config) ClientConfig() (*ssh.ClientConfig, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	authMethods, err := c.Auth.AuthMethods()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth method: %w", err)
	}

	hostKeyCallback, err := c.Auth.HostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to get host key callback: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            c.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.Timeout,
	}

	return clientConfig, nil
}
