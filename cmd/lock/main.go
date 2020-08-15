package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gcp-services/lock/backends"
	pb "github.com/gcp-services/lock/storage"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
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
	return s.db.Refresh(ctx, in)
}

func (s *service) Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	return s.db.Release(ctx, in)
}

func createService() (*service, error) {
	var svc service

	switch viper.GetString("backend") {
	case "spanner":
		path := viper.GetString("spanner.database")
		if path == "" {
			return nil, fmt.Errorf("no spanner database path specified")
		}

		sp, err := backends.NewSpanner(context.Background(), path)

		if err != nil {
			return nil, fmt.Errorf("failed to create spanner backend: %v", err)
		}
		svc.db = sp
	default:
		return nil, fmt.Errorf("backend not specified or invalid")
	}

	return &svc, nil
}

func config() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	pflag.Int("port", 9876, "listen port for gRPC connections")
	pflag.String("backend", "", "backend to use for locking")
	pflag.String("spanner.database", "", "spanner database path to use")
	pflag.String("bigtable.table", "", "bigtable table to use for locks")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	err := viper.ReadInConfig()
	switch err.(type) {
	case viper.ConfigFileNotFoundError:
		break
	default:
		return err
	}

	return nil
}

func main() {

	if err := config(); err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("port")))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	svc, err := createService()
	if err != nil {
		log.Fatalf("error creating service: %v", err)
	}

	pb.RegisterLockServiceServer(s, svc)

	log.Printf("starting server on port %d", viper.GetInt("port"))
	if err := s.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
