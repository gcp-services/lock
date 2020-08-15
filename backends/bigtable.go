package backends

import (
	"context"
	"encoding/binary"
	"log"
	"time"

	"cloud.google.com/go/bigtable"
	pb "github.com/gcp-services/lock/storage"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

// Bigtable service implements locks for a Bigtable backend.
type Bigtable struct {
	client *bigtable.Client
	admin  *bigtable.AdminClient
	table  *bigtable.Table
}

// NewBigtable create a new Bigtable based lock service.
func NewBigtable(ctx context.Context, project, instance string, opts ...option.ClientOption) (*Bigtable, error) {
	admin, err := bigtable.NewAdminClient(ctx, project, instance, opts...)
	if err != nil {
		return nil, err
	}

	client, err := bigtable.NewClient(ctx, project, instance, opts...)
	if err != nil {
		return nil, err
	}

	if err := admin.CreateTable(ctx, viper.GetString("bigtable.table")); err != nil {
		return nil, err
	}

	if err := admin.CreateColumnFamily(ctx, viper.GetString("bigtable.table"), "Locks"); err != nil {
		return nil, err
	}

	return &Bigtable{
		client: client,
		admin:  admin,
		table:  client.Open(viper.GetString("bigtable.table")),
	}, nil
}

func (b *Bigtable) applyLock(ctx context.Context, tag string, in *pb.TryLockRequest) (bool, error) {
	var filter bigtable.Filter
	if tag == "" {
		filter = bigtable.ChainFilters(
			bigtable.FamilyFilter("Locks"),
		)
	} else {
		filter = bigtable.ChainFilters(
			bigtable.FamilyFilter("Locks"),
			bigtable.ColumnFilter("Locks:etag"),
			bigtable.ValueFilter(tag),
		)
	}

	ts, err := ptypes.Timestamp(in.Lock.Expires)
	if err != nil {
		return false, err
	}

	timeBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBuffer, uint64(ts.Unix()))
	btime := bigtable.Time(ts)
	mut := bigtable.NewMutation()
	mut.Set("Locks", "etag", btime, []byte(uuid.New().String()))
	mut.Set("Locks", "owner", btime, []byte(in.Lock.Owner))
	mut.Set("Locks", "expires", btime, timeBuffer)
	var condMut *bigtable.Mutation
	if tag == "" {
		condMut = bigtable.NewCondMutation(filter, nil, mut)
	} else {
		condMut = bigtable.NewCondMutation(filter, mut, nil)
	}

	var matched bool
	if err := b.table.Apply(ctx, in.Lock.Uuid, condMut, bigtable.GetCondMutationResult(&matched)); err != nil {
		return false, err
	}

	if tag == "" {
		return !matched, nil
	}

	return matched, nil
}

// TryLock todo
func (b *Bigtable) TryLock(ctx context.Context, in *pb.TryLockRequest) (*pb.TryLockResponse, error) {

	// Try to apply the lock if the row doesn't exist.
	applied, err := b.applyLock(ctx, "", in)
	switch {
	case err != nil:
		return nil, err
	case applied:
		return &pb.TryLockResponse{}, nil
	}
	log.Printf("applied is %v\n", applied)
	/*
		// Try to apply the lock if the row doesn't exist.
		matched, err = b.applyLock(ctx, "", in)
		switch {
		case err != nil:
			return nil, err
		case matched:
			log.Printf("matched is %v\n", matched)
		}
	*/
	// Cond write, if etag == "", etag = new random uuid
	// ReadRow
	// Check time/owner/etc
	// Apply write via cond write along with a NEW etag value
	row, err := b.table.ReadRow(ctx, in.Lock.Uuid)
	if err != nil {
		return nil, err
	}

	// Row doesn't exist even though we (tried) to ensure it exists above.
	if len(row) == 0 {
		applied, err := b.applyLock(ctx, "", in)
		switch {
		case err != nil:
			return nil, err
		case applied:
			return &pb.TryLockResponse{}, nil
		case !applied:
			return nil, ErrLockBusy
		}
	}

	// Read in the stored lock values.
	values := make(map[string][]byte)
	for _, column := range row["Locks"] {
		values[column.Column] = column.Value
	}

	// Decode the expiry time for the lock.
	expires := time.Unix(int64(binary.BigEndian.Uint64(values["Locks:expires"])), 0)

	// Check if this lock can be applied, and try to do so.
	if time.Now().After(expires) {
		applied, err := b.applyLock(ctx, string(values["Locks:etag"]), in)
		switch {
		case err != nil:
			return nil, err
		case applied:
			return &pb.TryLockResponse{}, nil
		}
	}

	return nil, ErrLockBusy
}

// Lock will attempt to acquire a lock, blocking until a lock is acquired or until
// the timeout is met.
func (b *Bigtable) Lock(ctx context.Context, in *pb.LockRequest) (*pb.LockResponse, error) {
	return doLock(ctx, b, in)
}

func (b *Bigtable) Refresh(_ context.Context, _ *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	return nil, nil
	//panic("not implemented") // TODO: Implement
}

func (b *Bigtable) Release(_ context.Context, _ *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	return nil, nil
	//panic("not implemented") // TODO: Implement
}
