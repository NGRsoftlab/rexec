// Copyright © NGRSoftlab 2020-2025

package ssh

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestWithPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"too_large", 70000, true},
		{"valid", 2222, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Port: 22}
			op := WithPort(tc.port)
			err := op(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil && cfg.Port != tc.port {
				t.Errorf("Port=%d; want %d", cfg.Port, tc.port)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"negative", -5 * time.Second, true},
		{"zero", 0, true},
		{"valid", 10 * time.Second, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			op := WithTimeout(tc.timeout)
			err := op(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil && cfg.timeout != tc.timeout {
				t.Errorf("timeout=%v; want %v", cfg.timeout, tc.timeout)
			}
		})
	}
}

func TestWithRetry(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		interval time.Duration
		wantErr  bool
	}{
		{"neg_count", -1, 1 * time.Second, true},
		{"neg_interval", 1, -1 * time.Second, true},
		{"valid", 2, 500 * time.Millisecond, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			op := WithRetry(tc.count, tc.interval)
			err := op(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil && (cfg.retryCount != tc.count || cfg.retryInterval != tc.interval) {
				t.Errorf("retry=%d,%v; want %d,%v", cfg.retryCount, cfg.retryInterval, tc.count, tc.interval)
			}
		})
	}
}

func TestWithKeepAlive(t *testing.T) {
	tests := []struct {
		name    string
		ka      time.Duration
		wantErr bool
	}{
		{"zero", 0, true},
		{"valid", 15 * time.Second, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			op := WithKeepAlive(tc.ka)
			err := op(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil && cfg.keepAlive != tc.ka {
				t.Errorf("keepAlive=%v; want %v", cfg.keepAlive, tc.ka)
			}
		})
	}
}

func TestWithSudoPassword(t *testing.T) {
	tests := []struct {
		name, pwd string
		wantErr   bool
	}{
		{"empty", "", true},
		{"set", "p", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			op := WithSudoPassword(tc.pwd)
			err := op(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil && cfg.sudoPassword != tc.pwd {
				t.Errorf("sudoPassword=%q; want %q", cfg.sudoPassword, tc.pwd)
			}
		})
	}
}

func TestWithEnvVars(t *testing.T) {
	cfg := &Config{envVars: map[string]string{"A": "1"}}
	op := WithEnvVars(map[string]string{"B": "2", "A": "Z"})
	err := op(cfg)
	if err != nil {
		t.Fatalf("err=%v; want nil", err)
	}
	want := map[string]string{"A": "Z", "B": "2"}
	if !reflect.DeepEqual(cfg.envVars, want) {
		t.Errorf("envVars=%v; want %v", cfg.envVars, want)
	}
}

func TestWithKnownHosts(t *testing.T) {
	tests := []struct {
		name, path string
		wantErr    bool
	}{
		{"empty", "", true},
		{"not_exist", "no_file", true},
	}
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "kh")
	os.WriteFile(fpath, []byte(""), 0644)
	tests = append(tests, struct {
		name, path string
		wantErr    bool
	}{"valid", fpath, false})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			op := WithKnownHosts(tc.path)
			err := op(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil && cfg.knownHostsPath != tc.path {
				t.Errorf("knownHostsPath=%q; want %q", cfg.knownHostsPath, tc.path)
			}
		})
	}
}

func TestWithWorkdirAndMaxSessions(t *testing.T) {
	tests := []struct {
		name, wd string
		max      int
		wantErr  bool
	}{
		{"workdir_empty", "", 0, true},
		{"wd_set", "/tmp", 0, true},
		{"max_low", "/tmp", -1, true},
		{"max_high", "/tmp", 10, true},
		{"both_valid", "/tmp", 3, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			op1 := WithWorkdir(tc.wd)
			op2 := WithMaxSessions(tc.max)
			err1 := op1(cfg)
			err2 := op2(cfg)
			if tc.wd == "" {
				if err1 == nil {
					t.Errorf("expected error for empty wd")
				}
			} else if err1 != nil {
				t.Errorf("err1=%v; want nil", err1)
			}
			if (err2 != nil) != tc.wantErr {
				t.Errorf("err2=%v; wantErr=%v", err2, tc.wantErr)
			}
			if err2 == nil && cfg.maxSessions != tc.max {
				t.Errorf("maxSessions=%d; want %d", cfg.maxSessions, tc.max)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name, user, host string
		port             int
		wantErr          bool
	}{
		{"missing_user", "", "h", 22, true},
		{"missing_host", "u", "", 22, true},
		{"bad_port", "u", "h", 70000, true},
		{"valid", "u", "h", 22, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{User: tc.user, Host: tc.host, Port: tc.port}
			err := cfg.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v; wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestClientConfig(t *testing.T) {
	tests := []struct {
		name     string
		opts     []ConfigOption
		wantErr  bool
		wantAuth int
	}{
		{"no_auth", nil, true, 0},
		{"password", []ConfigOption{WithPasswordAuth("pw")}, false, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewConfig("u", "h", 22, tc.opts...)
			if err != nil {
				t.Fatalf("NewConfig err=%v", err)
			}
			cc, err := cfg.ClientConfig()
			if (err != nil) != tc.wantErr {
				t.Errorf("ClientConfig err=%v; wantErr=%v", err, tc.wantErr)
			}
			if err == nil {
				if cc.User != "u" {
					t.Errorf("User=%q; want u", cc.User)
				}
				if len(cc.Auth) != tc.wantAuth {
					t.Errorf("Auth count=%d; want %d", len(cc.Auth), tc.wantAuth)
				}
				if cc.Timeout != defaultTimeout {
					t.Errorf("Timeout=%v; want %v", cc.Timeout, defaultTimeout)
				}
			}
		})
	}
}

func TestHostKeyCallback(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "no.khs")
	cfg := &Config{knownHostsPath: bad}
	_, err := cfg.hostKeyCallback()
	if err == nil {
		t.Errorf("expected error for bad knownHostsPath")
	}

	cfg2 := &Config{}
	cb, err := cfg2.hostKeyCallback()
	if err != nil {
		t.Errorf("unexpected err=%v", err)
	}
	if cb == nil {
		t.Errorf("callback nil; want insecure ignore")
	}
	err = cb("host", nil, nil)
	if err != nil {
		t.Errorf("Insecure callback returned err=%v; want nil", err)
	}
}
