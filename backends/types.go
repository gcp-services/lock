package backends

import (
	"fmt"
)

var (
	// ErrLockBusy denotes a lock that has already been acquired.
	ErrLockBusy = fmt.Errorf("lock has already been acquired by another process")
	// ErrLockInvalidOwner denotes an unlock attempt by a caller that isn't the original owner.
	ErrLockInvalidOwner = fmt.Errorf("lock can not be unlocked by another process")
	// ErrLockInvalidRefresh denotes a refresh attempt that reduces the lock time.
	ErrLockInvalidRefresh = fmt.Errorf("lock can not be refreshed to a duration shorter than the current duration")
	// ErrLockNotFound denotes an attempt to refresh a lock that was not found.
	ErrLockNotFound = fmt.Errorf("lock not found")
)
