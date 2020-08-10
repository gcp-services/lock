package backends

import "context"

type Database interface {
	Connect(ctx context.Context, address string) error
}
