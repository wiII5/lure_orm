package lure_orm

import (
	"reflect"
	"testing"
	"time"
)

// testEntity is used for testing timestamp and diff logic.
// It mirrors the pattern used by generated entities.
type testEntity struct {
	ID        string    `spanner:"Id"`
	Name      string    `spanner:"Name"`
	Value     int64     `spanner:"Value"`
	CreatedAt time.Time `spanner:"CreatedAt"`
	UpdatedAt time.Time `spanner:"UpdatedAt"`
	Original  *testEntity
}

func (e *testEntity) SpannerPrimaryKeyColumns() []string {
	return []string{"Id"}
}

// ============================================================
// deref
// ============================================================

func TestDeref_NilPointer(t *testing.T) {
	var entity *testEntity
	rv := deref(reflect.ValueOf(entity))
	// dereferencing nil pointer yields invalid reflect.Value
	if rv.IsValid() {
		t.Error("expected invalid reflect.Value for nil pointer dereference")
	}
}

func TestDeref_NonPointer(t *testing.T) {
	entity := testEntity{ID: "1"}
	rv := deref(reflect.ValueOf(entity))
	if rv.Kind() != reflect.Struct {
		t.Errorf("expected Struct, got %v", rv.Kind())
	}
}

func TestDeref_SinglePointer(t *testing.T) {
	entity := &testEntity{ID: "1"}
	rv := deref(reflect.ValueOf(entity))
	if rv.Kind() != reflect.Struct {
		t.Errorf("expected Struct, got %v", rv.Kind())
	}
}

func TestDeref_DoublePointer(t *testing.T) {
	entity := &testEntity{ID: "1"}
	ptr := &entity
	rv := deref(reflect.ValueOf(ptr))
	if rv.Kind() != reflect.Struct {
		t.Errorf("expected Struct, got %v", rv.Kind())
	}
}

// ============================================================
// hasOriginal
// ============================================================

func TestHasOriginal_NilOriginal(t *testing.T) {
	entity := &testEntity{ID: "1", Original: nil}
	if hasOriginal(entity) {
		t.Error("expected hasOriginal=false when Original is nil")
	}
}

func TestHasOriginal_SetOriginal(t *testing.T) {
	entity := &testEntity{
		ID:       "1",
		Original: &testEntity{ID: "1"},
	}
	if !hasOriginal(entity) {
		t.Error("expected hasOriginal=true when Original is set")
	}
}

func TestHasOriginal_NoOriginalField(t *testing.T) {
	type noOriginal struct {
		ID string `spanner:"Id"`
	}
	entity := &noOriginal{ID: "1"}
	if hasOriginal(entity) {
		t.Error("expected hasOriginal=false for struct without Original field")
	}
}

func TestHasOriginal_NotAStruct(t *testing.T) {
	val := "not a struct"
	if hasOriginal(val) {
		t.Error("expected hasOriginal=false for non-struct")
	}
}

// ============================================================
// setInsertTimestamps
// ============================================================

func TestSetInsertTimestamps_SetsCreatedAtAndUpdatedAt(t *testing.T) {
	before := time.Now()
	entity := &testEntity{}
	setInsertTimestamps(entity)
	after := time.Now()

	if entity.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if entity.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if entity.CreatedAt.Before(before) || entity.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in [%v, %v]", entity.CreatedAt, before, after)
	}
	if entity.UpdatedAt.Before(before) || entity.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not in [%v, %v]", entity.UpdatedAt, before, after)
	}
}

func TestSetInsertTimestamps_NoTimestampFields(t *testing.T) {
	type noTimestamp struct{ ID string }
	// Should not panic
	setInsertTimestamps(&noTimestamp{ID: "1"})
}

func TestSetInsertTimestamps_NonPointer(t *testing.T) {
	entity := testEntity{}
	// Non-pointer: reflection cannot set fields, should not panic
	setInsertTimestamps(entity)
}

// ============================================================
// preserveCreatedAt
// ============================================================

func TestPreserveCreatedAt_CopiesFromOriginal(t *testing.T) {
	originalTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entity := &testEntity{
		CreatedAt: time.Time{}, // zero
		Original:  &testEntity{CreatedAt: originalTime},
	}
	preserveCreatedAt(entity)
	if !entity.CreatedAt.Equal(originalTime) {
		t.Errorf("expected CreatedAt=%v, got %v", originalTime, entity.CreatedAt)
	}
}

func TestPreserveCreatedAt_NoOriginal(t *testing.T) {
	entity := &testEntity{CreatedAt: time.Time{}}
	preserveCreatedAt(entity) // should not panic, CreatedAt stays zero
	if !entity.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to remain zero when no Original")
	}
}

func TestPreserveCreatedAt_NilOriginal(t *testing.T) {
	entity := &testEntity{Original: nil}
	preserveCreatedAt(entity) // should not panic
}

func TestPreserveCreatedAt_NoOriginalField(t *testing.T) {
	type noOriginal struct{ CreatedAt time.Time }
	entity := &noOriginal{}
	preserveCreatedAt(entity) // should not panic
}

// ============================================================
// setUpdateTimestamps
// ============================================================

func TestSetUpdateTimestamps_PreservesCreatedAtAndSetsUpdatedAt(t *testing.T) {
	originalCreated := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	entity := &testEntity{
		CreatedAt: time.Time{},
		Original:  &testEntity{CreatedAt: originalCreated},
	}
	before := time.Now()
	setUpdateTimestamps(entity)
	after := time.Now()

	if !entity.CreatedAt.Equal(originalCreated) {
		t.Errorf("expected CreatedAt preserved as %v, got %v", originalCreated, entity.CreatedAt)
	}
	if entity.UpdatedAt.Before(before) || entity.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not in [%v, %v]", entity.UpdatedAt, before, after)
	}
}

func TestSetUpdateTimestamps_NoOriginal(t *testing.T) {
	entity := &testEntity{}
	before := time.Now()
	setUpdateTimestamps(entity)
	after := time.Now()

	if entity.UpdatedAt.Before(before) || entity.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not in [%v, %v]", entity.UpdatedAt, before, after)
	}
}

// ============================================================
// diffColumns
// ============================================================

func TestDiffColumns_AlwaysIncludesPK(t *testing.T) {
	entity := &testEntity{
		ID:       "abc",
		Name:     "unchanged",
		Original: &testEntity{ID: "abc", Name: "unchanged"},
	}
	cols, vals := diffColumns(entity, []string{"Id"})

	if !containsStr(cols, "Id") {
		t.Errorf("expected PK 'Id' in columns, got %v", cols)
	}
	if !containsStr(cols, "UpdatedAt") {
		t.Errorf("expected 'UpdatedAt' in columns, got %v", cols)
	}

	pkIdx := indexOf(cols, "Id")
	if pkIdx >= 0 && vals[pkIdx] != "abc" {
		t.Errorf("expected PK value 'abc', got %v", vals[pkIdx])
	}
}

func TestDiffColumns_AlwaysIncludesUpdatedAt(t *testing.T) {
	entity := &testEntity{
		ID:       "abc",
		Original: &testEntity{ID: "abc"},
	}
	cols, _ := diffColumns(entity, []string{"Id"})
	if !containsStr(cols, "UpdatedAt") {
		t.Errorf("expected 'UpdatedAt' always included, got %v", cols)
	}
}

func TestDiffColumns_IncludesChangedField(t *testing.T) {
	entity := &testEntity{
		ID:       "abc",
		Name:     "new name",
		Original: &testEntity{ID: "abc", Name: "old name"},
	}
	cols, vals := diffColumns(entity, []string{"Id"})
	if !containsStr(cols, "Name") {
		t.Errorf("expected changed field 'Name' in columns, got %v", cols)
	}
	nameIdx := indexOf(cols, "Name")
	if nameIdx >= 0 && vals[nameIdx] != "new name" {
		t.Errorf("expected value 'new name', got %v", vals[nameIdx])
	}
}

func TestDiffColumns_ExcludesUnchangedField(t *testing.T) {
	entity := &testEntity{
		ID:       "abc",
		Name:     "same",
		Value:    42,
		Original: &testEntity{ID: "abc", Name: "same", Value: 42},
	}
	cols, _ := diffColumns(entity, []string{"Id"})
	if containsStr(cols, "Name") {
		t.Errorf("expected unchanged field 'Name' excluded, got %v", cols)
	}
	if containsStr(cols, "Value") {
		t.Errorf("expected unchanged field 'Value' excluded, got %v", cols)
	}
}

func TestDiffColumns_NoOriginal(t *testing.T) {
	entity := &testEntity{
		ID:   "abc",
		Name: "test",
	}
	// Without Original, all columns are included (no diff possible)
	cols, _ := diffColumns(entity, []string{"Id"})
	if !containsStr(cols, "Id") {
		t.Errorf("expected Id in columns, got %v", cols)
	}
	if !containsStr(cols, "Name") {
		t.Errorf("expected Name included when no Original, got %v", cols)
	}
}

func TestDiffColumns_MultipleChangedFields(t *testing.T) {
	entity := &testEntity{
		ID:       "abc",
		Name:     "new",
		Value:    99,
		Original: &testEntity{ID: "abc", Name: "old", Value: 1},
	}
	cols, _ := diffColumns(entity, []string{"Id"})
	if !containsStr(cols, "Name") {
		t.Errorf("expected Name in cols, got %v", cols)
	}
	if !containsStr(cols, "Value") {
		t.Errorf("expected Value in cols, got %v", cols)
	}
}

func TestDiffColumns_SkipsFieldsWithoutSpannerTag(t *testing.T) {
	entity := &testEntity{ID: "abc", Original: &testEntity{ID: "abc"}}
	cols, _ := diffColumns(entity, []string{"Id"})
	// "Original" field has no spanner tag, should not appear
	if containsStr(cols, "Original") {
		t.Errorf("expected Original field excluded, got %v", cols)
	}
}

// ============================================================
// helpers
// ============================================================

func containsStr(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

func indexOf(ss []string, target string) int {
	for i, s := range ss {
		if s == target {
			return i
		}
	}
	return -1
}
