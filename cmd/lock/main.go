package main

import (
	"context"
	"log"
	"net"

	"github.com/gcp-services/lock/backends"
	pb "github.com/gcp-services/lock/storage"
	"google.golang.org/grpc"
)

const (
	port = ":9876"
)

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

	pb.RegisterLockServiceServer(s, sp)
	log.Printf("starting server on port %s", port)
	if err := s.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
