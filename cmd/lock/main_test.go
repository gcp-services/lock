package main

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner/spannertest"
	"github.com/gcp-services/lock/backends"
	pb "github.com/gcp-services/lock/storage"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TestBackend struct {
	Name  string
	Flags map[string]interface{}
	Setup func() (pb.LockServiceServer, error)
}

var testBackends = []TestBackend{
	{
		Name: "spanner",
		Flags: map[string]interface{}{
			"spanner.database": "projects/test/instances/test/databases/test",
		},
		Setup: setupSpanner,
	},
}

func setupSpanner() (pb.LockServiceServer, error) {
	ctx := context.Background()

	server, err := spannertest.NewServer(":9010")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(server.Addr, grpc.WithInsecure())

	if err != nil {
		return nil, err
	}

	sp, err := backends.NewSpanner(ctx, viper.GetString("spanner.database"), option.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	if err := sp.CreateSchema(ctx, true); err != nil {
		return nil, err
	}

	return &service{
		db: sp,
	}, nil
}

func TestServer(t *testing.T) {
	if err := config(); err != nil {
		t.Fatalf("error setting up config: %v", err)
	}
	for _, backend := range testBackends {
		viper.Set("backend", backend.Name)
		for flagName, flagValue := range backend.Flags {
			viper.Set(flagName, flagValue)
		}
		svc, err := backend.Setup()

		if err != nil {
			t.Fatalf("unable to create service %s: %v", backend.Name, err)
		}
		ctx := context.Background()

		// Create a lock.
		expires := time.Now().Add(time.Second * 30)
		if _, err = svc.TryLock(ctx, &pb.TryLockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != nil {
			t.Fatalf("error trying to lock: %v", err)
		}
		// Attempt to relock.
		if _, err = svc.TryLock(ctx, &pb.TryLockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != backends.ErrLockBusy {
			t.Fatalf("expected lock to be busy, instead: %v", err)
		}

		// Atempt to unlock with the incorrect owner.
		if _, err = svc.Release(ctx, &pb.ReleaseRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "12345",
				Expires: timestamppb.New(expires),
			},
		}); err != backends.ErrLockInvalidOwner {
			t.Fatalf("expected lock to fail with invalid owner, instead: %v", err)
		}

		// Unlock with the correct owner.
		if _, err = svc.Release(ctx, &pb.ReleaseRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != nil {
			t.Fatalf("expected to unlock, instead: %v", err)
		}

		// Lock with a short expiry.
		expires = time.Now().Add(time.Millisecond * 100)
		if _, err = svc.TryLock(ctx, &pb.TryLockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != nil {
			t.Fatalf("error trying to lock: %v", err)
		}

		// Allow the lock to expire.
		time.Sleep(time.Millisecond * 200)

		// Overwrite the lock that has expired.
		if _, err = svc.TryLock(ctx, &pb.TryLockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != nil {
			t.Fatalf("error trying to lock: %v", err)
		}

		expires = time.Now().Add(time.Second * 3)
		// Lock again with a longer expiry.
		if _, err = svc.Lock(ctx, &pb.LockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != nil {
			t.Fatalf("error trying to lock: %v", err)
		}

		// Try to lock, while waiting only 100 ms.
		if _, err = svc.Lock(ctx, &pb.LockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
			Timeout: durationpb.New(time.Millisecond * 100),
		}); err != backends.ErrLockBusy {
			t.Fatalf("lock should be busy, instead: %v", err)
		}

		expires = time.Now().Add(time.Second * 10)
		// Try to lock and wait for the previous lock to expire.
		if _, err = svc.Lock(ctx, &pb.LockRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
			Timeout: durationpb.New(time.Second * 4),
		}); err != nil {
			t.Fatalf("error trying to lock: %v", err)
		}

		// Try to refresh the lock to an earlier expiry.
		expires = expires.Add(-time.Second)
		if _, err = svc.Refresh(ctx, &pb.RefreshRequest{
			Lock: &pb.Lock{
				Uuid:    "1234",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != backends.ErrLockInvalidRefresh {
			t.Fatalf("error refreshing lock: %v", err)
		}

		// Try to refresh a lock that doesn't exist.
		if _, err = svc.Refresh(ctx, &pb.RefreshRequest{
			Lock: &pb.Lock{
				Uuid:    "123456",
				Owner:   "1234",
				Expires: timestamppb.New(expires),
			},
		}); err != backends.ErrLockNotFound {
			t.Fatalf("error refreshing lock: %v", err)
		}
	}
}
