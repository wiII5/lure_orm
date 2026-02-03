package lure_orm

import (
	"context"

	"cloud.google.com/go/spanner"
)

// ReadRunner is an interface for read-only transactions.
type ReadRunner interface {
	Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
}

// ReadWriteRunner is an interface for read-write transactions.
type ReadWriteRunner interface {
	ReadRunner
	BufferWrite(ms []*spanner.Mutation) error
	Update(ctx context.Context, stmt spanner.Statement) (int64, error)
}
