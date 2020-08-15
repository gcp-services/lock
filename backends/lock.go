package backends

import (
	"context"
	"time"

	pb "github.com/gcp-services/lock/storage"
	"github.com/golang/protobuf/ptypes"
)

// doLock is a generic function for awaiting a lock. All backends should call this
// function in place of implementing Lock internally.
func doLock(ctx context.Context, svc pb.LockServiceServer, in *pb.LockRequest) (*pb.LockResponse, error) {
	start := time.Now()
	req := &pb.TryLockRequest{
		Lock: in.Lock,
	}

	_, err := svc.TryLock(ctx, req)

	switch err {
	case ErrLockBusy:
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
			return nil, ErrLockBusy
		}

		_, err := svc.TryLock(ctx, req)

		switch err {
		case ErrLockBusy:
			break
		case nil:
			return &pb.LockResponse{}, nil
		default:
			return nil, err
		}

		time.Sleep(time.Second)
	}
}
