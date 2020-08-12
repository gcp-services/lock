package backends

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner/spannertest"
	pb "github.com/gcp-services/lock/storage"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func createDatabase() {
}
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()

	server, err := spannertest.NewServer(":9010")
	if err != nil {
		t.Fatalf("unable to create test server: %v", err)
	}
	conn, err := grpc.Dial(server.Addr, grpc.WithInsecure())

	if err != nil {
		t.Fatalf("unable to dial server: %v", err)
	}

	sp, err := NewSpanner(ctx, "projects/test/instances/test/databases/test", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("unable to create client: %v", err)
	}

	if err := sp.CreateSchema(ctx, true); err != nil {
		t.Fatalf("unable to create database schema: %v", err)
	}

	// Create a lock.
	expires := time.Now().Add(time.Second * 30)
	if _, err = sp.TryLock(ctx, &pb.TryLockRequest{
		Lock: &pb.Lock{
			Uuid:    "1234",
			Owner:   "1234",
			Expires: timestamppb.New(expires),
		},
	}); err != nil {
		t.Fatalf("error trying to lock: %v", err)
	}

	// Attempt to relock.
	if _, err = sp.TryLock(ctx, &pb.TryLockRequest{
		Lock: &pb.Lock{
			Uuid:    "1234",
			Owner:   "1234",
			Expires: timestamppb.New(expires),
		},
	}); err != ErrLockBusy {
		t.Fatalf("expected lock to be busy, instead: %v", err)
	}

	// Atempt to unlock with the incorrect owner.
	if _, err = sp.Release(ctx, &pb.ReleaseRequest{
		Lock: &pb.Lock{
			Uuid:    "1234",
			Owner:   "12345",
			Expires: timestamppb.New(expires),
		},
	}); err != ErrLockInvalidOwner {
		t.Fatalf("expected lock to fail with invalid owner, instead: %v", err)
	}

	// Unlock with the correct owner.
	if _, err = sp.Release(ctx, &pb.ReleaseRequest{
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
	if _, err = sp.TryLock(ctx, &pb.TryLockRequest{
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
	if _, err = sp.TryLock(ctx, &pb.TryLockRequest{
		Lock: &pb.Lock{
			Uuid:    "1234",
			Owner:   "1234",
			Expires: timestamppb.New(expires),
		},
	}); err != nil {
		t.Fatalf("error trying to lock: %v", err)
	}
}
