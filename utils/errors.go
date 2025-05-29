package utils

import (
	"errors"
	"fmt"
)

var (
	ErrSessionNotOpen = errors.New("session not open")
	ErrClientNil      = errors.New("client is nil")
)

// ExitCodeMapper translates process exit codes into human-readable messages
type ExitCodeMapper struct {
	codes map[int]string
}

// NewDefaultExitCodeMapper returns an ExitCodeMapper initialized with common shell and system exit code messages
func NewDefaultExitCodeMapper() *ExitCodeMapper {
	return &ExitCodeMapper{codes: map[int]string{
		1:   "general error",
		2:   "invalid usage of shell builtins",
		126: "permission denied (cannot execute)",
		127: "command not found",
		128: "invalid exit argument",

		64: "command line usage error",
		65: "data format error",
		66: "cannot open input file",
		67: "address unknown",
		68: "host name unknown",
		69: "service unavailable",
		70: "internal software error",
		71: "system error",
		72: "critical OS file missing",
		73: "cannot create output file",
		74: "input/output error",
		75: "temporary failure, please retry",
		76: "remote protocol error",
		77: "permission denied",
		78: "configuration error",

		129: "hangup (SIGHUP)",
		130: "script terminated by Control-C",
		131: "quit (SIGQUIT)",
		132: "illegal instruction (SIGILL)",
		133: "trace/breakpoint trap (SIGTRAP)",
		134: "abort (SIGABRT)",
		135: "bus error (SIGBUS)",
		136: "floating point exception (SIGFPE)",
		137: "process killed (SIGKILL)",
		138: "user defined signal 1 (SIGUSR1)",
		139: "segmentation fault (SIGSEGV)",
		140: "user defined signal 2 (SIGUSR2)",
		141: "broken pipe (SIGPIPE)",
		142: "alarm clock (SIGALRM)",
		143: "terminated by signal (SIGTERM)",

		255: "ssh connection error or no exit status",
	}}
}

const maxSignal = 64 // highest signal number to consider

// Lookup returns a descriptive message for the given exit code.
// If the code is in the predefined map, that message is used.
// Codes 129–(128+maxSignal) map to "killed by signal N".
// All others default to "exit <code>".
func (em *ExitCodeMapper) Lookup(code int) string {
	if msg, ok := em.codes[code]; ok {
		return msg
	}

	if code > 128 && code <= 128+maxSignal {
		return fmt.Sprintf("killed by signal %d", code-128)
	}
	return fmt.Sprintf("exit %d", code)
}
