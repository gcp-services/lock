package backends

import (
	"context"

	"cloud.google.com/go/spanner"
	pb "github.com/gcp-services/lock/storage"
	"google.golang.org/api/option"
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
	return nil, nil
}

// Lock will attempt to acquire a lock, blocking until a lock is acquired or until
// the timeout is met.
func (s *Spanner) Lock(ctx context.Context, in *pb.LockRequest) (*pb.LockResponse, error) {
	return nil, nil
}

// Release will release a lock that was previously acquired.
func (s *Spanner) Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	return nil, nil
}
