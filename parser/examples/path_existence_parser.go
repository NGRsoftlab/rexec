// Copyright © NGRSoftlab 2020-2025

package examples

import (
	"errors"
	"strings"

	"github.com/ngrsoftlab/rexec/parser"
)

// PathExistence implements parser.Parser to convert stdout into a boolean
type PathExistence struct{}

// Parse checks rawResult.Stdout for "true" (case-insensitive) and sets dst.(*bool) accordingly.
// Returns an error if dst is not a *bool
func (p *PathExistence) Parse(rawResult *parser.RawResult, dst any) error {
	exists, ok := dst.(*bool)
	if !ok {
		return errors.New("dst is not a bool")
	}
	*exists = strings.TrimSpace(strings.ToLower(rawResult.Stdout)) == "true"
	return nil
}
