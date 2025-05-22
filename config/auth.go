package config

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHAuth holds the private key, password, and agent flags for authentication
type SSHAuth struct {
	password   string // optional password
	keyPath    string // optional path to private key file
	keyBytes   []byte // optional data of private key file
	passphrase string // optional, if private key is encrypted
	useAgent   bool   // optional
}

// withPassword configures password-based auth
func (a *SSHAuth) withPassword(password string) error {
	if len(password) == 0 {
		return fmt.Errorf("password empty")
	}
	a.password = password
	return nil
}

// withPrivateKeyPath configures file-based key authentication
func (a *SSHAuth) withPrivateKeyPath(path, passphrase string) error {
	if len(path) == 0 {
		return fmt.Errorf("private key path empty")
	}
	a.keyPath = path
	a.passphrase = passphrase
	return nil
}

// withPrivateKeyBytes configures in-memory key authentication
func (a *SSHAuth) withPrivateKeyBytes(privateKey []byte, passphrase string) error {
	if len(privateKey) == 0 {
		return fmt.Errorf("private key bytes empty")
	}
	a.keyBytes = privateKey
	a.passphrase = passphrase
	return nil
}

// withAgent adds SSH agent auth. (Unix systems only)
func (a *SSHAuth) withAgent() error {
	a.useAgent = true
	return nil
}

// buildAgentAuth dials the SSH agent and returns its ssh.AuthMethod (Unix systems only)
func (a *SSHAuth) buildAgentAuth() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("dial agent: %w", err)
	}
	ag := agent.NewClient(conn)
	return ssh.PublicKeysCallback(ag.Signers), nil
}

// authMethods returns a slice of ssh.AuthMethod in the order:
// agent → private key (file or bytes) → password.
// Returns an error if none succeed
func (a *SSHAuth) authMethods() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if a.useAgent {
		if m, err := a.buildAgentAuth(); err == nil {
			methods = append(methods, m)
		}
	}
	if len(a.keyPath) > 0 {
		keyData, fileErr := os.ReadFile(a.keyPath)
		if fileErr != nil {
			return nil, fmt.Errorf("read key file: %w", fileErr)
		}
		signer, err := parseSigner(keyData, a.passphrase)
		if err != nil {
			return methods, fmt.Errorf("read key file: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if len(a.keyBytes) > 0 {
		signer, err := parseSigner(a.keyBytes, a.passphrase)
		if err != nil {
			return methods, fmt.Errorf("read key bytes: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if len(a.password) > 0 {
		methods = append(methods, ssh.Password(a.password))
	}

	if len(methods) == 0 {
		return methods, fmt.Errorf("no valid auth methods available")
	}

	return methods, nil
}

// parseSigner parses PEM private key, decrypting with passphrase if any.
func parseSigner(data []byte, passphrase string) (ssh.Signer, error) {
	if len(passphrase) > 0 {
		return ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	}
	return ssh.ParsePrivateKey(data)
}
