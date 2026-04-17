package lure_orm

import (
	"context"
	"errors"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ErrNoRows is returned when QueryRow doesn't find any rows.
var ErrNoRows = errors.New("lure_orm: no rows in result set")

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

// QueryAll executes a raw SQL statement and returns all matching rows.
func QueryAll[T any](ctx context.Context, txn ReadRunner, stmt spanner.Statement) ([]*T, error) {
	return iterateAll[T](txn.Query(ctx, stmt))
}

// QueryOne executes a raw SQL statement and returns the first matching row, or nil if none found.
func QueryOne[T any](ctx context.Context, txn ReadRunner, stmt spanner.Statement) (*T, error) {
	return iterateOne[T](txn.Query(ctx, stmt))
}

// QueryCount executes a raw SQL statement and returns the count result.
func QueryCount(ctx context.Context, txn ReadRunner, stmt spanner.Statement) (int64, error) {
	return iterateCount(txn.Query(ctx, stmt))
}

// QueryExists executes a raw SQL statement and returns true if any rows match.
func QueryExists(ctx context.Context, txn ReadRunner, stmt spanner.Statement) (bool, error) {
	return iterateExists(txn.Query(ctx, stmt))
}

// InsertStruct buffers an insert mutation from a struct.
// Automatically sets CreatedAt and UpdatedAt to time.Now().
func InsertStruct(ctx context.Context, txn ReadWriteRunner, table string, v interface{}) error {
	setInsertTimestamps(v)
	m, err := spanner.InsertStruct(table, v)
	if err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// UpdateStruct buffers an update mutation from a struct.
// Automatically preserves CreatedAt from Original and sets UpdatedAt to time.Now().
func UpdateStruct(ctx context.Context, txn ReadWriteRunner, table string, v interface{}) error {
	setUpdateTimestamps(v)
	m, err := spanner.UpdateStruct(table, v)
	if err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// InsertOrUpdateStruct buffers an insert-or-update mutation from a struct.
// Automatically sets UpdatedAt to time.Now(). Sets CreatedAt only if zero.
func InsertOrUpdateStruct(ctx context.Context, txn ReadWriteRunner, table string, v interface{}) error {
	setInsertTimestamps(v)
	m, err := spanner.InsertOrUpdateStruct(table, v)
	if err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// InsertStructMulti buffers multiple insert mutations from structs.
// Automatically sets CreatedAt and UpdatedAt to time.Now() on each item.
func InsertStructMulti[T any](ctx context.Context, txn ReadWriteRunner, table string, items []*T) error {
	if len(items) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(items))
	for _, item := range items {
		setInsertTimestamps(item)
		m, err := spanner.InsertStruct(table, item)
		if err != nil {
			return err
		}
		mutations = append(mutations, m)
	}
	return txn.BufferWrite(mutations)
}

// UpdateStructMulti buffers multiple update mutations from structs.
// Automatically preserves CreatedAt from Original and sets UpdatedAt to time.Now() on each item.
func UpdateStructMulti[T any](ctx context.Context, txn ReadWriteRunner, table string, items []*T) error {
	if len(items) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(items))
	for _, item := range items {
		setUpdateTimestamps(item)
		m, err := spanner.UpdateStruct(table, item)
		if err != nil {
			return err
		}
		mutations = append(mutations, m)
	}
	return txn.BufferWrite(mutations)
}

// InsertOrUpdateStructMulti buffers multiple insert-or-update mutations from structs.
func InsertOrUpdateStructMulti[T any](ctx context.Context, txn ReadWriteRunner, table string, items []*T) error {
	if len(items) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(items))
	for _, item := range items {
		m, err := spanner.InsertOrUpdateStruct(table, item)
		if err != nil {
			return err
		}
		mutations = append(mutations, m)
	}
	return txn.BufferWrite(mutations)
}

// DeleteMulti buffers multiple delete mutations.
func DeleteMulti(ctx context.Context, txn ReadWriteRunner, table string, keys []spanner.Key) error {
	if len(keys) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(keys))
	for _, key := range keys {
		mutations = append(mutations, spanner.Delete(table, key))
	}
	return txn.BufferWrite(mutations)
}

// ReadRow reads a single row by primary key.
func ReadRow[T any](ctx context.Context, txn ReadRunner, table string, key spanner.Key, columns []string) (*T, error) {
	row, err := txn.ReadRow(ctx, table, key, columns)
	if err != nil {
		return nil, err
	}
	var item T
	if err := row.ToStruct(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

// ExecUpdate executes a DML statement and returns the number of affected rows.
func ExecUpdate(ctx context.Context, txn ReadWriteRunner, stmt spanner.Statement) (int64, error) {
	return txn.Update(ctx, stmt)
}

// ============================================================
// Non-generic query functions for manual scanning
// ============================================================

// ExecuteQuery executes a SQL statement and returns an iterator for manual scanning.
// The caller is responsible for calling iter.Stop() when done.
func ExecuteQuery(ctx context.Context, txn ReadRunner, stmt spanner.Statement) *spanner.RowIterator {
	return txn.Query(ctx, stmt)
}

// Row wraps a spanner.Row to provide a Scan method similar to database/sql.
type Row struct {
	row *spanner.Row
	err error
}

// Scan reads columns from the row into the provided pointers.
// Returns ErrNoRows if no row was found.
func (r *Row) Scan(ptrs ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.row == nil {
		return ErrNoRows
	}
	return r.row.Columns(ptrs...)
}

// ToStruct reads the row into a struct.
// Returns ErrNoRows if no row was found.
func (r *Row) ToStruct(p interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.row == nil {
		return ErrNoRows
	}
	return r.row.ToStruct(p)
}

// Err returns any error that occurred during the query.
func (r *Row) Err() error {
	return r.err
}

// QueryRow executes a SQL statement and returns a single Row for scanning.
// If no rows are found, Scan will return ErrNoRows.
func QueryRow(ctx context.Context, txn ReadRunner, stmt spanner.Statement) *Row {
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return &Row{row: nil, err: nil} // Scan will return ErrNoRows
		}
		return &Row{row: nil, err: err}
	}
	return &Row{row: row, err: nil}
}

// IterateRows iterates over all rows and calls the provided function for each row.
// Stops iteration if the function returns an error.
func IterateRows(ctx context.Context, txn ReadRunner, stmt spanner.Statement, fn func(*spanner.Row) error) error {
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				return nil
			}
			return err
		}
		if err := fn(row); err != nil {
			return err
		}
	}
}
