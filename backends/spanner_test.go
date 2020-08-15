package backends

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner/spannertest"
	pb "github.com/gcp-services/lock/storage"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

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

	sp, err := NewSpanner(ctx, viper.GetString("spanner.database"), option.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	if err := sp.CreateSchema(ctx, true); err != nil {
		return nil, err
	}

	return sp, nil
}

func TestSpanner(t *testing.T) {
	testServer(t, &testBackend{
		Name: "spanner",
		Flags: map[string]interface{}{
			"spanner.database": "projects/test/instances/test/databases/test",
		},
		Setup: setupSpanner,
	})
}
