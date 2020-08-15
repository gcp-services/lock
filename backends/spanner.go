package backends

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	admin "cloud.google.com/go/spanner/admin/database/apiv1"
	pb "github.com/gcp-services/lock/storage"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	"google.golang.org/grpc/codes"
)

// Spanner is an implementation of the Lock server that uses Spanner as a backing store.
type Spanner struct {
	client       *spanner.Client
	admin        *admin.DatabaseAdminClient
	databasePath string
	databaseName string
	instance     string
}

// NewSpanner creates a new connection to Spanner and returns the Spanner object.
func NewSpanner(ctx context.Context, database string, opts ...option.ClientOption) (*Spanner, error) {
	instance := strings.Join(strings.Split(database, "/")[:4], "/")
	databaseName := strings.Split(database, "/")[5]

	adminClient, err := admin.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	client, err := spanner.NewClient(ctx, database, opts...)
	if err != nil {
		return nil, err
	}

	sp := &Spanner{
		client:       client,
		admin:        adminClient,
		databasePath: database,
		databaseName: databaseName,
		instance:     instance,
	}

	return sp, nil
}

// CreateSchema creates the schema for this database.
func (s *Spanner) CreateSchema(ctx context.Context, testing bool) error {
	if testing {
		op, err := s.admin.UpdateDatabaseDdl(ctx, &database.UpdateDatabaseDdlRequest{
			Database: s.databaseName,
			Statements: []string{
				`CREATE TABLE Locks (
					uuid STRING(MAX) NOT NULL,
					owner STRING(MAX) NOT NULL,
					expires TIMESTAMP NOT NULL,
					) PRIMARY KEY (uuid)`,
			},
		})
		if err != nil {
			return err
		}

		err = op.Wait(ctx)
		return err
	}

	_, err := s.admin.GetDatabase(ctx, &database.GetDatabaseRequest{
		Name: s.databasePath,
	})

	switch {
	case spanner.ErrCode(err) == codes.NotFound:
		break
	default:
		return err
	}

	op, err := s.admin.CreateDatabase(ctx, &database.CreateDatabaseRequest{
		Parent:          s.instance,
		CreateStatement: fmt.Sprintf("CREATE DATABASE %s", s.databaseName),
		ExtraStatements: []string{
			`CREATE TABLE Locks (
						uuid STRING(MAX) NOT NULL,
						owner STRING(MAX) NOT NULL,
						expires TIMESTAMP NOT NULL,
						) PRIMARY KEY (uuid)`,
		},
	})

	if err != nil {
		return err
	}

	if _, err := op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Spanner) applyLock(txn *spanner.ReadWriteTransaction, in *pb.TryLockRequest) error {
	ts, err := ptypes.Timestamp(in.Lock.Expires)
	if err != nil {
		return err
	}
	m := spanner.InsertOrUpdate("Locks", []string{"uuid", "owner", "expires"}, []interface{}{
		in.Lock.Uuid,
		in.Lock.Owner,
		ts,
	})
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// TryLock will attempt to acquire a lock. If a lock can not be acquired, the function
// will return immediately with the reason for lock acquisition failure.
func (s *Spanner) TryLock(ctx context.Context, in *pb.TryLockRequest) (*pb.TryLockResponse, error) {
	if _, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Locks", spanner.Key{in.Lock.GetUuid()}, []string{"uuid", "owner", "expires"})
		switch {
		case spanner.ErrCode(err) == codes.NotFound:
			return s.applyLock(txn, in)
		case err != nil:
			return err
		}

		// A lock entry was found, validate it.
		readLock := pb.Lock{}
		var expires time.Time
		if err = row.Columns(&readLock.Uuid, &readLock.Owner, &expires); err != nil {
			return err
		}

		// Check if this lock is expired and has not been refreshed. Claim this lock
		// if the lock has expired.
		if time.Now().After(expires) {
			return s.applyLock(txn, in)
		}
		return ErrLockBusy
	}); err != nil {
		return nil, err
	}

	return &pb.TryLockResponse{}, nil
}

// Lock will attempt to acquire a lock, blocking until a lock is acquired or until
// the timeout is met.
func (s *Spanner) Lock(ctx context.Context, in *pb.LockRequest) (*pb.LockResponse, error) {
	return doLock(ctx, s, in)
}

// Refresh will refresh a lock lease and extend the time a valid lock is held.
func (s *Spanner) Refresh(ctx context.Context, in *pb.RefreshRequest) (*pb.RefreshResponse, error) {

	if _, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Locks", spanner.Key{in.Lock.GetUuid()}, []string{"uuid", "owner", "expires"})
		switch {
		case spanner.ErrCode(err) == codes.NotFound:
			return ErrLockNotFound
		case err != nil:
			return err
		}

		// A lock entry was found, validate it.
		readLock := pb.Lock{}
		var expires time.Time
		if err = row.Columns(&readLock.Uuid, &readLock.Owner, &expires); err != nil {
			return err
		}

		// Check if the refresh time is before the current expiry time.
		ts, err := ptypes.Timestamp(in.Lock.Expires)
		if err != nil {
			return err
		}

		if ts.Before(expires) {
			return ErrLockInvalidRefresh
		}

		return s.applyLock(txn, &pb.TryLockRequest{
			Lock: in.Lock,
		})

	}); err != nil {
		return nil, err
	}

	return &pb.RefreshResponse{}, nil
}

// Release will release a lock that was previously acquired.
func (s *Spanner) Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	if _, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Locks", spanner.Key{in.Lock.GetUuid()}, []string{"uuid", "owner", "expires"})
		switch {
		case spanner.ErrCode(err) == codes.NotFound:
			return nil
		case err != nil:
			return err
		}

		// Lock found, read it.
		readLock := pb.Lock{}
		var expires time.Time
		if err = row.Columns(&readLock.Uuid, &readLock.Owner, &expires); err != nil {
			return err
		}

		if readLock.Owner != in.Lock.Owner {
			return ErrLockInvalidOwner
		}
		m := spanner.Delete("Locks", spanner.Key{in.Lock.GetUuid()})
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		return nil, err
	}
	return nil, nil
}
