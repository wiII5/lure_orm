package lure_orm

import (
	"reflect"
	"time"
)

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

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}
