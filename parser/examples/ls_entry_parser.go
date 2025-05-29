package examples

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ngrsoftlab/rexec/parser"
)

// LsEntry represents one line of `ls -la` output
type LsEntry struct {
	Permissions string // raw permission string (e.g. "-rw-r--r--")
	Links       int    // number of hard links
	Owner       string // file owner name
	Group       string // file group name
	Size        int64  // file size in bytes
	Month       string // month of last modification
	Day         string // day of month of last modification
	TimeOrYear  string // time or year of last modification
	Name        string // file or directory name
}

// ParsePermissions converts Permissions into an os.FileMode value.
// It interprets the first character as file type and the next
// nine as owner/group/other permission bits.
func (e *LsEntry) ParsePermissions() (os.FileMode, error) {
	permRE := regexp.MustCompile(`^([dlbcp\-s][rwx-]{9})`)
	m := permRE.FindStringSubmatch(e.Permissions)
	if len(m) < 2 {
		return 0, fmt.Errorf("invalid perm string: %q", e.Permissions)
	}
	perm := m[1] // 10 chars, e.g. "-rw-r--r--"

	var mode os.FileMode
	// file type
	switch perm[0] {
	case 'd':

		mode |= os.ModeDir
	case 'l':
		mode |= os.ModeSymlink
	case 'b':
		mode |= os.ModeDevice | os.ModeCharDevice
	case 'c':
		mode |= os.ModeDevice
	case 'p':
		mode |= os.ModeNamedPipe
	case 's':
		mode |= os.ModeSocket
	}
	// owner/group/other
	perms := []struct {
		char byte
		bit  os.FileMode
	}{
		{'r', 0400}, {'w', 0200}, {'x', 0100},
		{'r', 0040}, {'w', 0020}, {'x', 0010},
		{'r', 0004}, {'w', 0002}, {'x', 0001},
	}
	for i, p := range perms {
		if perm[i+1] == p.char {
			mode |= p.bit
		}
	}
	return mode, nil
}

// LsParser implements parser.Parser for `ls -la` output
type LsParser struct{}

// Parse reads raw.Stdout, skips the "total" header line,
// splits each data line into fields, and appends parsed LsEntry items
// to dst.(*[]LsEntry). Returns an error if dst is not the correct type
func (p *LsParser) Parse(raw *parser.RawResult, dst any) error {
	slicePtr, ok := dst.(*[]LsEntry)
	if !ok {
		return fmt.Errorf("dst must be *[]LsEntry")
	}

	lines := strings.Split(strings.TrimSpace(raw.Stdout), "\n")
	var entries []LsEntry
	for i, line := range lines {
		if i == 0 && strings.HasPrefix(line, "total") || strings.HasPrefix(line, "Total") || strings.HasPrefix(line,
			"итого") || strings.HasPrefix(line, "Итого") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 9 {
			continue
		}
		perm := parts[0]
		if len(perm) > 10 {
			perm = perm[:10]
		}

		links, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid links %q: %w", parts[1], err)
		}

		size, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid size %q: %w", parts[4], err)
		}

		name := strings.Join(parts[8:], " ")

		entries = append(entries, LsEntry{
			Permissions: parts[0],
			Links:       links,
			Owner:       parts[2],
			Group:       parts[3],
			Size:        size,
			Month:       parts[5],
			Day:         parts[6],
			TimeOrYear:  parts[7],
			Name:        name,
		})
	}
	*slicePtr = entries
	return nil
}
