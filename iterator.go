package lure_orm

import (
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func iterateAll[T any](iter *spanner.RowIterator) ([]*T, error) {
	defer iter.Stop()

	var results []*T
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("lure_orm: iteration error: %w", err)
		}
		var item T
		if err := row.ToStruct(&item); err != nil {
			return nil, fmt.Errorf("lure_orm: scan error: %w", err)
		}
		results = append(results, &item)
	}
	return results, nil
}

func iterateOne[T any](iter *spanner.RowIterator) (*T, error) {
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lure_orm: iteration error: %w", err)
	}
	var item T
	if err := row.ToStruct(&item); err != nil {
		return nil, fmt.Errorf("lure_orm: scan error: %w", err)
	}
	return &item, nil
}

func iterateCount(iter *spanner.RowIterator) (int64, error) {
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("lure_orm: count error: %w", err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("lure_orm: count scan error: %w", err)
	}
	return count, nil
}
