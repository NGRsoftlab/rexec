package config

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHAuthMethod defines supported SSH authentication types
type SSHAuthMethod int

const (
	SSHAuthPassword SSHAuthMethod = iota
	SSHAuthPrivateKeyPath
	SSHAuthPrivateKeyBytes
	SSHAuthAgent
)

// Auth defines SSH authentication configuration
type Auth struct {
	Password       string // optional password
	KeyPath        string // optional path to private key file
	KeyBytes       []byte // optional data of private key file
	Passphrase     string // optional, if private key is encrypted
	KnownHostsPath string // optional path to known_hosts file to check real SSH host keys
	Methods        []SSHAuthMethod
}

// withPassword adds password authentication
func (a *Auth) withPassword(password string) *Auth {
	a.Password = password
	a.Methods = append(a.Methods, SSHAuthPassword)
	return a
}

// withPrivateKeyPath adds private key file authentication
func (a *Auth) withPrivateKeyPath(path, passphrase string) *Auth {
	a.KeyPath = path
	a.Passphrase = passphrase
	a.Methods = append(a.Methods, SSHAuthPrivateKeyPath)
	return a
}

// withPrivateKeyBytes adds private key bytes authentication
func (a *Auth) withPrivateKeyBytes(privateKey []byte, passphrase string) *Auth {
	a.KeyBytes = privateKey
	a.Passphrase = passphrase
	a.Methods = append(a.Methods, SSHAuthPrivateKeyBytes)
	return a
}

// withAgent adds SSH agent authentication. (Unix systems only)
func (a *Auth) withAgent() *Auth {
	a.Methods = append(a.Methods, SSHAuthAgent)
	return a
}

// AuthMethods returns prepared []ssh.AuthMethod based on enabled methods
func (a *Auth) AuthMethods() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	for _, method := range a.Methods {
		authMethod, err := a.authMethodForType(method)
		if err != nil {
			return nil, err
		}
		methods = append(methods, authMethod)
	}
	if len(methods) == 0 {
		return nil, errors.New("no valid auth methods configured")
	}
	return methods, nil
}

// HostKeyCallback returns a host key verification callback
func (a *Auth) HostKeyCallback() (ssh.HostKeyCallback, error) {
	if a.KnownHostsPath != "" {
		return knownhosts.New(a.KnownHostsPath)
	}
	return ssh.InsecureIgnoreHostKey(), nil
}

func (a *Auth) authMethodForType(method SSHAuthMethod) (ssh.AuthMethod, error) {
	switch method {
	case SSHAuthPassword:
		return a.passwordAuthMethod()
	case SSHAuthPrivateKeyPath:
		return a.privateKeyPathAuthMethod()
	case SSHAuthPrivateKeyBytes:
		return a.privateKeyBytesAuthMethod()
	case SSHAuthAgent:
		return a.signerFromAgent()

	default:
		return nil, fmt.Errorf("unknown auth method: %v", method)
	}
}

// passwordAuthMethod returns ssh.AuthMethod based on password
func (a *Auth) passwordAuthMethod() (ssh.AuthMethod, error) {
	if a.Password == "" {
		return nil, errors.New("password is empty")
	}
	return ssh.Password(a.Password), nil
}

// privateKeyPathAuthMethod returns ssh.AuthMethod based on private key path
func (a *Auth) privateKeyPathAuthMethod() (ssh.AuthMethod, error) {
	if a.KeyPath == "" {
		return nil, errors.New("private key path is empty")
	}
	signer, err := a.signerFromKeyFile(a.KeyPath, a.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from file: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

// privateKeyBytesAuthMethod returns ssh.AuthMethod based on private key bytes
func (a *Auth) privateKeyBytesAuthMethod() (ssh.AuthMethod, error) {
	if len(a.KeyBytes) == 0 {
		return nil, errors.New("private key bytes are empty")
	}
	signer, err := a.signerFromKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from bytes: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

// signerFromKeyFile returns signer from private key file. If Auth contain Passphrase - decrypt key
func (a *Auth) signerFromKeyFile(path, passphrase string) (ssh.Signer, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(a.Passphrase) > 0 {
		return ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	}
	return ssh.ParsePrivateKey(keyData)
}

// signerFromKeyBytes returns signer from private key bytes. If Auth contain Passphrase - decrypt key
func (a *Auth) signerFromKeyBytes() (ssh.Signer, error) {
	if len(a.Passphrase) > 0 {
		return ssh.ParsePrivateKeyWithPassphrase(a.KeyBytes, []byte(a.Passphrase))
	}
	return ssh.ParsePrivateKey(a.KeyBytes)
}

// signerFromAgent returns ssh.AuthMethod based on agent (Unix systems only)
func (a *Auth) signerFromAgent() (ssh.AuthMethod, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, fmt.Errorf("could not find ssh agent: %w", err)
	}
	ag := agent.NewClient(conn)
	return ssh.PublicKeysCallback(ag.Signers), nil
}
