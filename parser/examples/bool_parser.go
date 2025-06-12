// Copyright © NGRSoftlab 2020-2025

package examples

import (
	"fmt"
	"strings"

	"github.com/ngrsoftlab/rexec/parser"
)

// BoolParser implements parser.Parser for any boolean-like stdout.
type BoolParser struct{}

// Parse reads raw.Stdout, trims spaces, lower-cases it, and sets dst.(*bool).
// Returns an error if dst is not *bool or the text isn’t recognized.
func (p *BoolParser) Parse(raw *parser.RawResult, dst any) error {
	bprt, ok := dst.(*bool)
	if !ok {
		return fmt.Errorf("dst must be *bool, got %T", dst)
	}
	text := strings.TrimSpace(strings.ToLower(raw.Stdout))
	switch text {
	case "1", "t", "true", "yes", "y", "on":
		*bprt = true
	case "0", "f", "false", "no", "n", "off":
		*bprt = false
	default:
		return fmt.Errorf("unrecognized bool value: %q", raw.Stdout)
	}
	return nil
}
