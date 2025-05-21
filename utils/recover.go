package utils

import (
	"fmt"
	"runtime/debug"
)

// Recover recovers from a panic return the error.
func Recover() (err error) {
	if r := recover(); r != nil {
		err = fmt.Errorf("recovering from panic: %v\n%s", r, debug.Stack())
	}
	return err
}
