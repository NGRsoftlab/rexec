package examples

import (
	"errors"
	"strings"

	"github.com/ngrsoftlab/rexec/parser"
)

type PathExistence struct {
}

func (p *PathExistence) Parse(rawResult *parser.RawResult, dst any) error {
	exists, ok := dst.(*bool)
	if !ok {
		return errors.New("dst is not a bool")
	}
	*exists = strings.TrimSpace(strings.ToLower(rawResult.Stdout)) == "true"
	return nil
}
