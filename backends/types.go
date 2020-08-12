package backends

import (
	"fmt"
)

var (
	// ErrLockBusy denotes a lock that has already been acquired.
	ErrLockBusy = fmt.Errorf("lock has already been acquired by another process")
	// ErrLockInvalidOwner denotes an unlock attempt by a caller that isn't the original owner.
	ErrLockInvalidOwner = fmt.Errorf("lock can not be unlocked by another process")
)
