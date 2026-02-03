package lure_orm

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Find executes the query and returns all matching rows.
func Find[T any](ctx context.Context, txn ReadRunner, q *Query) ([]*T, error) {
	stmt, err := q.ToStatement()
	if err != nil {
		return nil, err
	}
	return iterateAll[T](txn.Query(ctx, stmt))
}

// FindOne executes the query and returns the first matching row, or nil if none found.
func FindOne[T any](ctx context.Context, txn ReadRunner, q *Query) (*T, error) {
	stmt, err := q.ToStatement()
	if err != nil {
		return nil, err
	}
	return iterateOne[T](txn.Query(ctx, stmt))
}

// Count executes a COUNT(*) query and returns the count.
func Count(ctx context.Context, txn ReadRunner, q *Query) (int64, error) {
	stmt, err := q.ToCountStatement()
	if err != nil {
		return 0, err
	}
	return iterateCount(txn.Query(ctx, stmt))
}

// Exists checks if any rows match the query.
func Exists(ctx context.Context, txn ReadRunner, q *Query) (bool, error) {
	count, err := Count(ctx, txn, q)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Insert buffers an insert mutation.
func Insert(ctx context.Context, txn ReadWriteRunner, table string, columns []string, values []interface{}) error {
	m := spanner.InsertOrUpdate(table, columns, values)
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// Update buffers an update mutation for the given key.
func Update(ctx context.Context, txn ReadWriteRunner, table string, columns []string, values []interface{}, key spanner.Key) error {
	_ = key // key is embedded in the columns/values for Spanner mutations
	m := spanner.InsertOrUpdate(table, columns, values)
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// Delete buffers a delete mutation for the given key.
func Delete(ctx context.Context, txn ReadWriteRunner, table string, key spanner.Key) error {
	m := spanner.Delete(table, key)
	return txn.BufferWrite([]*spanner.Mutation{m})
}
