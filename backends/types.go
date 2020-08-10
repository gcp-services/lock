package backends

import (
	"context"
	"fmt"
)

type Database interface {
	Connect(ctx context.Context, address string) error
}

var (
	ErrLockBusy = fmt.Errorf("lock has already been acquired by another process")
)
