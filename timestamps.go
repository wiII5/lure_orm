package lure_orm

import (
	"reflect"
	"time"
)

// EntityWithPK is implemented by generated entities and provides Spanner primary key column names
// for diff-based partial update logic.
type EntityWithPK interface {
	SpannerPrimaryKeyColumns() []string
}

// diffColumns returns columns/values for a partial UPDATE mutation.
// It always includes PK columns and UpdatedAt, plus any column whose value differs from Original.
func diffColumns(v interface{}, pkColumns []string) ([]string, []interface{}) {
	rv := deref(reflect.ValueOf(v))
	pkSet := make(map[string]bool, len(pkColumns))
	for _, pk := range pkColumns {
		pkSet[pk] = true
	}

	origField := rv.FieldByName(fieldOriginal)
	var orig reflect.Value
	if origField.IsValid() && !origField.IsNil() {
		orig = deref(origField)
	}

	rt := rv.Type()
	var columns []string
	var values []interface{}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		spannerTag := field.Tag.Get("spanner")
		if spannerTag == "" || spannerTag == "-" {
			continue
		}
		fv := rv.Field(i).Interface()

		// Always include PK columns
		if pkSet[spannerTag] {
			columns = append(columns, spannerTag)
			values = append(values, fv)
			continue
		}

		// Always include UpdatedAt
		if field.Name == fieldUpdatedAt {
			columns = append(columns, spannerTag)
			values = append(values, fv)
			continue
		}

		// Include if changed from Original
		if orig.IsValid() {
			origFV := orig.FieldByName(field.Name)
			if origFV.IsValid() && reflect.DeepEqual(fv, origFV.Interface()) {
				continue // unchanged
			}
		}

		columns = append(columns, spannerTag)
		values = append(values, fv)
	}

	return columns, values
}

const (
	fieldCreatedAt = "CreatedAt"
	fieldUpdatedAt = "UpdatedAt"
	fieldOriginal  = "Original"
)

// setInsertTimestamps sets CreatedAt and UpdatedAt to time.Now() via reflection.
// Called automatically by InsertStruct / InsertStructMulti.
func setInsertTimestamps(v interface{}) {
	now := time.Now()
	setTimeField(v, fieldCreatedAt, now)
	setTimeField(v, fieldUpdatedAt, now)
}

// setUpdateTimestamps preserves CreatedAt from Original (if present) and sets
// UpdatedAt to time.Now() via reflection.
// Called automatically by UpdateStruct / UpdateStructMulti.
func setUpdateTimestamps(v interface{}) {
	preserveCreatedAt(v)
	setTimeField(v, fieldUpdatedAt, time.Now())
}

// preserveCreatedAt copies CreatedAt from the Original field (DB snapshot)
// to the entity itself, preventing accidental overwrites on UPDATE.
func preserveCreatedAt(v interface{}) {
	rv := deref(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return
	}
	origField := rv.FieldByName(fieldOriginal)
	if !origField.IsValid() || origField.IsNil() {
		return
	}
	orig := deref(origField)
	origCreatedAt := orig.FieldByName(fieldCreatedAt)
	if !origCreatedAt.IsValid() {
		return
	}
	dst := rv.FieldByName(fieldCreatedAt)
	if dst.IsValid() && dst.CanSet() && dst.Type() == origCreatedAt.Type() {
		dst.Set(origCreatedAt)
	}
}

// setTimeField sets a time.Time field by name on a struct (or pointer to struct).
func setTimeField(v interface{}, fieldName string, val time.Time) {
	rv := deref(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return
	}
	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return
	}
	if f.Type() != reflect.TypeOf(time.Time{}) {
		return
	}
	f.Set(reflect.ValueOf(val))
}

// hasOriginal reports whether v has a non-nil Original field.
func hasOriginal(v interface{}) bool {
	rv := deref(reflect.ValueOf(v))
	if rv.Kind() != reflect.Struct {
		return false
	}
	f := rv.FieldByName(fieldOriginal)
	return f.IsValid() && !f.IsNil()
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}
