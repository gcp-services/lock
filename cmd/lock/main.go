package main

import (
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

	pb.RegisterLockServiceServer(s, &backends.Spanner{})
	log.Printf("starting server on port %s", port)
	if err := s.Serve(l); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
