package sender

import (
	"context"
	"io"
	"os"
)

// Sender handles file transfer.
type Sender interface {
	Send(ctx context.Context, src io.Reader, destPath string, mode os.FileMode) error
}
