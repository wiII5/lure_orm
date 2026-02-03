package lure_orm

import (
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// IterateAll は複数行のクエリ結果を構造体スライスに変換する
func IterateAll[T any](iter *spanner.RowIterator) ([]*T, error) {
	return iterateAll[T](iter)
}

// IterateOne は単一行のクエリ結果を構造体に変換する
func IterateOne[T any](iter *spanner.RowIterator) (*T, error) {
	return iterateOne[T](iter)
}

// IterateCount はカウントクエリの結果を取得する
func IterateCount(iter *spanner.RowIterator) (int64, error) {
	return iterateCount(iter)
}

// IterateExists は存在チェッククエリの結果を取得する
func IterateExists(iter *spanner.RowIterator) (bool, error) {
	return iterateExists(iter)
}

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
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lure_orm: count error: %w", err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("lure_orm: count scan error: %w", err)
	}
	return count, nil
}

func iterateExists(iter *spanner.RowIterator) (bool, error) {
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lure_orm: exists error: %w", err)
	}
	return true, nil
}
