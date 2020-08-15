package backends

import (
	"context"
	"testing"

	"cloud.google.com/go/bigtable/bttest"
	pb "github.com/gcp-services/lock/storage"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func setupBigtable() (pb.LockServiceServer, error) {
	ctx := context.Background()
	server, err := bttest.NewServer(":9011")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(server.Addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	sp, err := NewBigtable(ctx, "test", "test", option.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	return sp, nil
}
func TestBigtable(t *testing.T) {
	testServer(t, &testBackend{
		Name: "bigtable",
		Flags: map[string]interface{}{
			"bigtable.table": "test",
		},
		Setup: setupBigtable,
	})
}
