package utils

import (
	"reflect"
	"time"

	"cloud.google.com/go/spanner"
)

// TableName extracts the table name from a type parameter.
// Usage: utils.TableName[MyModel]()
func TableName[T any]() string {
	var zero T
	return reflect.TypeOf(zero).Name()
}

// ColumnNames extracts column names from a struct type using spanner tags.
// Returns a comma-separated string of column names.
func ColumnNames[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var names string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Tag.Get(TagSpanner)
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}
		if names != "" {
			names += ", "
		}
		names += name
	}
	return names
}

// GetFieldTag returns the lure_orm tag value for a struct field.
func GetFieldTag[T any](fieldName string) string {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	field, ok := t.FieldByName(fieldName)
	if !ok {
		return ""
	}
	return field.Tag.Get(TagLureORM)
}

// FindDeleteTimeColumn returns the column name marked with delete_time tag.
func FindDeleteTimeColumn[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get(TagLureORM) == TagDeleteTime {
			name := field.Tag.Get(TagSpanner)
			if name == "" {
				name = field.Name
			}
			return name
		}
	}
	return ""
}

// FindPrimaryKeyColumn returns the column name marked with primary tag.
func FindPrimaryKeyColumn[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get(TagLureORM) == TagPrimary {
			name := field.Tag.Get(TagSpanner)
			if name == "" {
				name = field.Name
			}
			return name
		}
	}
	return ""
}

// IsNullable checks if a value is a Spanner nullable type.
func IsNullable(v any) bool {
	switch v.(type) {
	case spanner.NullInt64, *spanner.NullInt64,
		spanner.NullFloat64, *spanner.NullFloat64,
		spanner.NullString, *spanner.NullString,
		spanner.NullDate, *spanner.NullDate,
		spanner.NullTime, *spanner.NullTime,
		spanner.NullBool, *spanner.NullBool:
		return true
	}
	return false
}

// IsTime checks if a value is a time type.
func IsTime(v any) bool {
	switch v.(type) {
	case time.Time, *time.Time, spanner.NullTime, *spanner.NullTime:
		return true
	}
	return false
}

// IsZeroTime checks if a time value is zero.
func IsZeroTime(v any) bool {
	switch t := v.(type) {
	case time.Time:
		return t.IsZero()
	case *time.Time:
		return t == nil || t.IsZero()
	case spanner.NullTime:
		return !t.Valid || t.Time.IsZero()
	case *spanner.NullTime:
		return t == nil || !t.Valid || t.Time.IsZero()
	}
	return false
}

// Contains checks if a slice contains a value.
func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
