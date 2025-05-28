package ssh

import (
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// auth holds the private key, password, and agent flags for authentication
type auth struct {
	password   string // optional password
	keyPath    string // optional path to private key file
	keyBytes   []byte // optional data of private key file
	passphrase string // optional, if private key is encrypted
	useAgent   bool   // optional
}

// withPassword configures password-based auth
func (a *auth) withPassword(password string) error {
	if len(password) == 0 {
		return fmt.Errorf("password empty")
	}
	a.password = password
	return nil
}

// withPrivateKeyPath configures file-based key authentication
func (a *auth) withPrivateKeyPath(path, passphrase string) error {
	if len(path) == 0 {
		return fmt.Errorf("private key path empty")
	}
	a.keyPath = path
	a.passphrase = passphrase
	return nil
}

// withPrivateKeyBytes configures in-memory key authentication
func (a *auth) withPrivateKeyBytes(privateKey []byte, passphrase string) error {
	if len(privateKey) == 0 {
		return fmt.Errorf("private key bytes empty")
	}
	a.keyBytes = privateKey
	a.passphrase = passphrase
	return nil
}

// withAgent adds SSH agent auth. (Unix systems only)
func (a *auth) withAgent() error {
	a.useAgent = true
	return nil
}

// buildAgentAuth dials the SSH agent and returns its ssh.AuthMethod (Unix systems only)
func (a *auth) buildAgentAuth() (ssh.AuthMethod, error) {
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
func (a *auth) authMethods() ([]ssh.AuthMethod, error) {
	methods := make([]ssh.AuthMethod, 0, 4)
	var errors []string

	if a.useAgent {
		if m, err := a.buildAgentAuth(); err != nil {
			errors = append(errors, fmt.Sprintf("agent: %v", err))
		} else {
			methods = append(methods, m)
		}
	}

	if a.keyPath != "" {
		keyData, fileErr := os.ReadFile(a.keyPath)
		if fileErr != nil {
			errors = append(errors, fmt.Sprintf("read key file: %v", fileErr))
		} else {
			signer, err := parseSigner(keyData, a.passphrase)
			if err != nil {
				errors = append(errors, fmt.Sprintf("read key file: %v", err))
			} else {
				methods = append(methods, ssh.PublicKeys(signer))
			}
		}

	}

	if len(a.keyBytes) > 0 {
		signer, err := parseSigner(a.keyBytes, a.passphrase)
		if err != nil {
			errors = append(errors, fmt.Sprintf("read key bytes: %v", err))
		} else {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}

	if a.password != "" {
		// INFO: Some OSs (like OpenSuse) require PAM authentication and the password will not authenticate.
		// The best solution is to use another method

		// Keyboard-interactive fallback
		methods = append(methods, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range questions {
					answers[i] = a.password
				}
				return answers, nil
			},
		),
			ssh.Password(a.password),
		)
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no valid auth methods available: %s", strings.Join(errors, "; "))
	}

	return methods, nil
}

// parseSigner parses PEM private key, decrypting with passphrase if any.
func parseSigner(data []byte, passphrase string) (ssh.Signer, error) {
	if len(passphrase) > 0 {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
		if err != nil && strings.Contains(err.Error(), "key is not password protected") {
			return ssh.ParsePrivateKey(data)
		}
		return signer, err
	}
	return ssh.ParsePrivateKey(data)
}
