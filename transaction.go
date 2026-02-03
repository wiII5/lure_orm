package lure_orm

import (
	"context"

	"cloud.google.com/go/spanner"
)

// ReadRunner is an interface for read-only transactions.
// Both *spanner.ReadOnlyTransaction and *spanner.ReadWriteTransaction implement this.
type ReadRunner interface {
	Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
	Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) *spanner.RowIterator
	ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
}

// ReadWriteRunner is an interface for read-write transactions.
// *spanner.ReadWriteTransaction implements this.
type ReadWriteRunner interface {
	ReadRunner
	BufferWrite(ms []*spanner.Mutation) error
	Update(ctx context.Context, stmt spanner.Statement) (int64, error)
}
