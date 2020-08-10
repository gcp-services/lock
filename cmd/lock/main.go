package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/gcp-services/lock/backends"
	pb "github.com/gcp-services/lock/storage"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

const (
	port = ":9876"
)

type service struct {
	db pb.LockServiceServer
}

func (s *service) TryLock(ctx context.Context, in *pb.TryLockRequest) (*pb.TryLockResponse, error) {
	return s.db.TryLock(ctx, in)
}

func (s *service) Lock(ctx context.Context, in *pb.LockRequest) (*pb.LockResponse, error) {
	start := time.Now()
	req := &pb.TryLockRequest{
		Lock: in.Lock,
	}

	_, err := s.db.TryLock(ctx, req)

	switch err {
	case backends.ErrLockBusy:
		break
	case nil:
		// TODO(lobato): craft a response here.
		return &pb.LockResponse{}, nil
	}

	dur, err := ptypes.Duration(in.Timeout)
	if err != nil {
		return nil, err
	}

	for {
		// Check if the maximum wait duration has expired.
		if time.Now().After(start.Add(dur)) {
			return nil, backends.ErrLockBusy
		}

		_, err := s.db.TryLock(ctx, req)

		switch err {
		case backends.ErrLockBusy:
			break
		case nil:
			return &pb.LockResponse{}, nil
		default:
			return nil, err
		}

		time.Sleep(time.Second)
	}
}

func (s *service) Refresh(ctx context.Context, in *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	return nil, nil
}
func (s *service) Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	return nil, nil
}

func main() {
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// TODO(lobato): Create config
	sp, err := backends.NewSpanner(context.Background(), "projects/test/instances/test/databases/test")
	if err != nil {
		log.Fatalf("failed to create spanner backend: %v", err)
	}

	pb.RegisterLockServiceServer(s, &service{
		db: sp,
	})

	log.Printf("starting server on port %s", port)
	if err := s.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
