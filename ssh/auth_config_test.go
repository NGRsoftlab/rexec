// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"os"
	"strings"
	"testing"
)

func TestWithPassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"nonempty", "secret", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &auth{}
			err := a.withPassword(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tc.wantErr)
			}
			if err == nil && a.password != tc.input {
				t.Errorf("password = %q; want %q", a.password, tc.input)
			}
		})
	}
}

func TestWithPrivateKeyPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		passphrase string
		wantErr    bool
	}{
		{"empty_path", "", "", true},
		{"valid_path", "/tmp/key", "pwd", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &auth{}
			err := a.withPrivateKeyPath(tc.path, tc.passphrase)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tc.wantErr)
			}
			if err == nil {
				if a.keyPath != tc.path || a.passphrase != tc.passphrase {
					t.Errorf("state = %v,%v; want %v,%v", a.keyPath, a.passphrase, tc.path, tc.passphrase)
				}
			}
		})
	}
}

func TestWithPrivateKeyBytes(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		passphrase string
		wantErr    bool
	}{
		{"empty_bytes", []byte{}, "", true},
		{"nonempty_bytes", []byte{1, 2, 3}, "pwd", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &auth{}
			err := a.withPrivateKeyBytes(tc.data, tc.passphrase)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tc.wantErr)
			}
			if err == nil {
				if !strings.EqualFold(string(a.keyBytes), string(tc.data)) || a.passphrase != tc.passphrase {
					t.Errorf("state bytes=%v,pass=%v; want %v,%v", a.keyBytes, a.passphrase, tc.data, tc.passphrase)
				}
			}
		})
	}
}

func TestWithAgent(t *testing.T) {
	a := &auth{}
	err := a.withAgent()
	if err != nil {
		t.Fatalf("withAgent error = %v; want nil", err)
	}
	if !a.useAgent {
		t.Errorf("useAgent = false; want true")
	}
}

func TestBuildAgentAuth(t *testing.T) {
	os.Unsetenv("SSH_AUTH_SOCK")
	a := &auth{useAgent: true}
	_, err := a.buildAgentAuth()
	if err == nil || !strings.Contains(err.Error(), "dial agent") {
		t.Errorf("err = %v; want dial agent error", err)
	}
}

func TestAuthMethods(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(a *auth)
		wantCount int
		wantErr   string
	}{
		{"none", func(a *auth) {}, 0, "no valid auth methods"},
		{"password", func(a *auth) { a.password = "pw" }, 2, ""},
		{"agent_fail", func(a *auth) { a.useAgent = true }, 0, "agent:"},
		{"key_bytes_fail", func(a *auth) { a.keyBytes = []byte("bad") }, 0, "read key bytes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &auth{}
			tc.setup(a)
			methods, err := a.authMethods()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("err = %v; want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err = %v", err)
			}
			if len(methods) != tc.wantCount {
				t.Errorf("methods count = %d; want %d", len(methods), tc.wantCount)
			}
		})
	}
}

func TestParseSigner(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		passphrase string
		wantErr    bool
	}{
		{"empty_data", []byte{}, "", true},
		{"invalid_data", []byte("bad"), "", true},
		{"invalid_with_pass", []byte("bad"), "pwd", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSigner(tc.data, tc.passphrase)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v; wantErr %v", err, tc.wantErr)
			}
		})
	}
}
