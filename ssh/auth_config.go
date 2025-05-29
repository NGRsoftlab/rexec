package ssh

import (
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// auth holds credentials and flags for SSH authentication methods
type auth struct {
	password   string // optional: password for password-based auth
	keyPath    string // optional: filesystem path to private key
	keyBytes   []byte // optional: in-memory private key data
	passphrase string // optional: passphrase for encrypted private key
	useAgent   bool   // optional: whether to try SSH agent auth
}

// withPassword enables password-based authentication
func (a *auth) withPassword(password string) error {
	if len(password) == 0 {
		return fmt.Errorf("password empty")
	}
	a.password = password
	return nil
}

// withPrivateKeyPath sets up authentication using a private key file
func (a *auth) withPrivateKeyPath(path, passphrase string) error {
	if len(path) == 0 {
		return fmt.Errorf("private key path empty")
	}
	a.keyPath = path
	a.passphrase = passphrase
	return nil
}

// withPrivateKeyBytes sets up authentication using in-memory private key data
func (a *auth) withPrivateKeyBytes(privateKey []byte, passphrase string) error {
	if len(privateKey) == 0 {
		return fmt.Errorf("private key bytes empty")
	}
	a.keyBytes = privateKey
	a.passphrase = passphrase
	return nil
}

// withAgent enables SSH agent authentication (UNIX only)
func (a *auth) withAgent() error {
	a.useAgent = true
	return nil
}

// buildAgentAuth connects to the SSH agent and returns its AuthMethod
func (a *auth) buildAgentAuth() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("dial agent: %w", err)
	}
	ag := agent.NewClient(conn)
	return ssh.PublicKeysCallback(ag.Signers), nil
}

// authMethods collects available ssh.AuthMethod in order of preference:
// agent → private key (file, then bytes) → password (keyboard-interactive + password).
// Returns an error if no methods are valid.
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

// parseSigner parses a PEM-encoded private key, decrypting if a passphrase is provided
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
