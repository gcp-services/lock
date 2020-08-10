package backends

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	pb "github.com/gcp-services/lock/storage"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
)

// Spanner is an implementation of the Lock server that uses Spanner as a backing store.
type Spanner struct {
	client *spanner.Client
}

// NewSpanner creates a new connection to Spanner and returns the Spanner object.
func NewSpanner(ctx context.Context, database string, opts ...option.ClientOption) (*Spanner, error) {
	client, err := spanner.NewClient(ctx, database, opts...)
	if err != nil {
		return nil, err
	}

	return &Spanner{
		client: client,
	}, nil
}

// TryLock will attempt to acquire a lock. If a lock can not be acquired, the function
// will return immediately with the reason for lock acquisition failure.
func (s *Spanner) TryLock(ctx context.Context, in *pb.TryLockRequest) (*pb.TryLockResponse, error) {
	if _, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Locks", spanner.Key{"uuid"}, []string{"owner", "expires"})
		switch {
		case spanner.ErrCode(err) == codes.NotFound:
			// TODO(lobato): apply lock
			break
		case err != nil:
			return err
		}

		// A lock entry was found, validate it.
		readLock := pb.Lock{}
		row.Columns(&readLock.Uuid, &readLock.Owner, &readLock.Expires)
		expire, err := ptypes.Timestamp(readLock.Expires)
		if err != nil {
			return err
		}

		// Check if this lock is expired and has not been refreshed. Claim this lock
		// if the lock has expired.
		if time.Now().After(expire) {
			// TODO(lobato): apply lock
			return nil
		}

		return ErrLockBusy
	}); err != nil {
		return nil, err
	}

	return nil, nil
}

// Lock will attempt to acquire a lock, blocking until a lock is acquired or until
// the timeout is met.
func (s *Spanner) Lock(ctx context.Context, in *pb.LockRequest) (*pb.LockResponse, error) {
	return nil, nil
}

// Refresh will refresh a lock lease and extend the time a valid lock is held.
func (s *Spanner) Refresh(ctx context.Context, in *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	return nil, nil
}

// Release will release a lock that was previously acquired.
func (s *Spanner) Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	return nil, nil
}
