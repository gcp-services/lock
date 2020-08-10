package backends

import (
	"fmt"
)

var (
	// ErrLockBusy denotes a lock that has already been acquired.
	ErrLockBusy = fmt.Errorf("lock has already been acquired by another process")
)
