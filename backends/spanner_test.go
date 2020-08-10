package backends

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner/spannertest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func TestNewSpanner(t *testing.T) {
	server, err := spannertest.NewServer(":19999")
	if err != nil {
		t.Fatalf("unable to setup server: %v", err)
	}
	ctx := context.Background()
	conn, err := grpc.Dial(server.Addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("unable to dial server: %v", err)
	}
	_, err = NewSpanner(ctx, "projects/test/instances/test/databases/test", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("unable to create client: %v", err)
	}
}
