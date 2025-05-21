package utils

import (
	"errors"
	"fmt"
)

var (
	ErrSessionClosed = errors.New("session closed")
	ErrContextDone   = errors.New("context cancelled or deadline exceeded")
)

type ExitCodeMapper struct {
	codes map[int]string
}

func NewDefaultExitCodeMapper() *ExitCodeMapper {
	return &ExitCodeMapper{codes: map[int]string{
		// General errors
		1: "general error",
		2: "invalid usage of shell builtins",

		// BSD-style sysexits
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

		// Shell-specific
		126: "permission denied (cannot execute)",
		127: "command not found",
		128: "invalid exit argument",

		// Signal-based exits (128 + N)
		130: "script terminated by Control-C",
		137: "process killed (SIGKILL)",
		139: "segmentation fault (SIGSEGV)",
		143: "terminated by signal (SIGTERM)",
	}}
}

// Lookup returns a message for code, or “exit <code>” if unknown.
// Also handles 129–255 as “killed by signal <N>”.
func (em *ExitCodeMapper) Lookup(code int) string {
	if msg, ok := em.codes[code]; ok {
		return msg
	}

	switch {
	case code > 127 && code < 256:
		sig := code - 128
		return fmt.Sprintf("killed by signal %d", sig)
	default:
		return fmt.Sprintf("exit %d", code)
	}
}
