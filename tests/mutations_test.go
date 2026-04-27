package tests

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	lure_orm "github.com/wiII5/lure_orm"
)

// ============================================================
// Mock ReadWriteRunner
// ============================================================

type mockTxn struct {
	mutations []*spanner.Mutation
}

func (m *mockTxn) BufferWrite(ms []*spanner.Mutation) error {
	m.mutations = append(m.mutations, ms...)
	return nil
}

func (m *mockTxn) Query(_ context.Context, _ spanner.Statement) *spanner.RowIterator {
	return nil
}

func (m *mockTxn) Read(_ context.Context, _ string, _ spanner.KeySet, _ []string) *spanner.RowIterator {
	return nil
}

func (m *mockTxn) ReadRow(_ context.Context, _ string, _ spanner.Key, _ []string) (*spanner.Row, error) {
	return nil, nil
}

func (m *mockTxn) Update(_ context.Context, _ spanner.Statement) (int64, error) {
	return 0, nil
}

// ============================================================
// Test entity with CreatedAt/UpdatedAt and EntityWithPK support
// ============================================================

type TrackedEntity struct {
	TrackID   string    `spanner:"TrackId"`
	Name      string    `spanner:"Name"`
	Score     int64     `spanner:"Score"`
	CreatedAt time.Time `spanner:"CreatedAt"`
	UpdatedAt time.Time `spanner:"UpdatedAt"`
	Original  *TrackedEntity
}

func (e *TrackedEntity) SpannerPrimaryKeyColumns() []string {
	return []string{"TrackId"}
}

const trackedTable = "Tracked"

// ============================================================
// InsertStruct
// ============================================================

func TestInsertStruct_BuffersOneMutation(t *testing.T) {
	txn := &mockTxn{}
	entity := &TrackedEntity{TrackID: "id1", Name: "test"}
	if err := lure_orm.InsertStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(txn.mutations))
	}
}

func TestInsertStruct_SetsCreatedAtAndUpdatedAt(t *testing.T) {
	txn := &mockTxn{}
	entity := &TrackedEntity{TrackID: "id1", Name: "test"}
	before := time.Now()
	if err := lure_orm.InsertStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if entity.CreatedAt.IsZero() || entity.CreatedAt.Before(before) || entity.CreatedAt.After(after) {
		t.Errorf("unexpected CreatedAt: %v", entity.CreatedAt)
	}
	if entity.UpdatedAt.IsZero() || entity.UpdatedAt.Before(before) || entity.UpdatedAt.After(after) {
		t.Errorf("unexpected UpdatedAt: %v", entity.UpdatedAt)
	}
}

// ============================================================
// UpdateStruct
// ============================================================

func TestUpdateStruct_BuffersOneMutation(t *testing.T) {
	txn := &mockTxn{}
	original := &TrackedEntity{TrackID: "id1", Name: "old"}
	entity := &TrackedEntity{TrackID: "id1", Name: "new", Original: original}
	if err := lure_orm.UpdateStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(txn.mutations))
	}
}

func TestUpdateStruct_SetsUpdatedAt(t *testing.T) {
	txn := &mockTxn{}
	original := &TrackedEntity{TrackID: "id1", Name: "old"}
	entity := &TrackedEntity{TrackID: "id1", Name: "new", Original: original}
	before := time.Now()
	if err := lure_orm.UpdateStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if entity.UpdatedAt.Before(before) || entity.UpdatedAt.After(after) {
		t.Errorf("unexpected UpdatedAt: %v", entity.UpdatedAt)
	}
}

func TestUpdateStruct_PreservesCreatedAt(t *testing.T) {
	txn := &mockTxn{}
	originalCreated := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	original := &TrackedEntity{TrackID: "id1", CreatedAt: originalCreated}
	entity := &TrackedEntity{TrackID: "id1", Name: "new", Original: original}
	if err := lure_orm.UpdateStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !entity.CreatedAt.Equal(originalCreated) {
		t.Errorf("expected CreatedAt preserved as %v, got %v", originalCreated, entity.CreatedAt)
	}
}

func TestUpdateStruct_WithoutEntityWithPK_UsesFullUpdate(t *testing.T) {
	type simpleEntity struct {
		ID        string    `spanner:"Id"`
		Name      string    `spanner:"Name"`
		CreatedAt time.Time `spanner:"CreatedAt"`
		UpdatedAt time.Time `spanner:"UpdatedAt"`
		Original  *struct {
			ID        string    `spanner:"Id"`
			Name      string    `spanner:"Name"`
			CreatedAt time.Time `spanner:"CreatedAt"`
			UpdatedAt time.Time `spanner:"UpdatedAt"`
		}
	}
	txn := &mockTxn{}
	entity := &simpleEntity{ID: "1", Name: "test"}
	if err := lure_orm.UpdateStruct(t.Context(), txn, "Simple", entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(txn.mutations))
	}
}

// ============================================================
// InsertOrUpdateStruct
// ============================================================

func TestInsertOrUpdateStruct_InsertWhenNoOriginal(t *testing.T) {
	txn := &mockTxn{}
	entity := &TrackedEntity{TrackID: "id1", Name: "new"}
	if err := lure_orm.InsertOrUpdateStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(txn.mutations))
	}
	// Should set CreatedAt (insert path)
	if entity.CreatedAt.IsZero() {
		t.Error("expected CreatedAt set on insert path")
	}
}

func TestInsertOrUpdateStruct_UpdateWhenOriginalSet(t *testing.T) {
	txn := &mockTxn{}
	original := &TrackedEntity{TrackID: "id1", Name: "old"}
	entity := &TrackedEntity{TrackID: "id1", Name: "new", Original: original}
	if err := lure_orm.InsertOrUpdateStruct(t.Context(), txn, trackedTable, entity); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(txn.mutations))
	}
	// Should set UpdatedAt (update path)
	if entity.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt set on update path")
	}
}

// ============================================================
// InsertStructMulti
// ============================================================

func TestInsertStructMulti_EmptySlice(t *testing.T) {
	txn := &mockTxn{}
	if err := lure_orm.InsertStructMulti(t.Context(), txn, trackedTable, []*TrackedEntity{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 0 {
		t.Errorf("expected 0 mutations for empty slice, got %d", len(txn.mutations))
	}
}

func TestInsertStructMulti_MultipleMutations(t *testing.T) {
	txn := &mockTxn{}
	entities := []*TrackedEntity{
		{TrackID: "id1", Name: "one"},
		{TrackID: "id2", Name: "two"},
		{TrackID: "id3", Name: "three"},
	}
	if err := lure_orm.InsertStructMulti(t.Context(), txn, trackedTable, entities); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 3 {
		t.Errorf("expected 3 mutations, got %d", len(txn.mutations))
	}
	for i, entity := range entities {
		if entity.CreatedAt.IsZero() {
			t.Errorf("entity[%d]: expected CreatedAt set", i)
		}
		if entity.UpdatedAt.IsZero() {
			t.Errorf("entity[%d]: expected UpdatedAt set", i)
		}
	}
}

// ============================================================
// UpdateStructMulti
// ============================================================

func TestUpdateStructMulti_EmptySlice(t *testing.T) {
	txn := &mockTxn{}
	if err := lure_orm.UpdateStructMulti(t.Context(), txn, trackedTable, []*TrackedEntity{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 0 {
		t.Errorf("expected 0 mutations, got %d", len(txn.mutations))
	}
}

func TestUpdateStructMulti_MultipleMutations(t *testing.T) {
	txn := &mockTxn{}
	original1 := &TrackedEntity{TrackID: "id1", Name: "old1"}
	original2 := &TrackedEntity{TrackID: "id2", Name: "old2"}
	entities := []*TrackedEntity{
		{TrackID: "id1", Name: "new1", Original: original1},
		{TrackID: "id2", Name: "new2", Original: original2},
	}
	if err := lure_orm.UpdateStructMulti(t.Context(), txn, trackedTable, entities); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(txn.mutations))
	}
	for i, entity := range entities {
		if entity.UpdatedAt.IsZero() {
			t.Errorf("entity[%d]: expected UpdatedAt set", i)
		}
	}
}

// ============================================================
// InsertOrUpdateStructMulti
// ============================================================

func TestInsertOrUpdateStructMulti_EmptySlice(t *testing.T) {
	txn := &mockTxn{}
	if err := lure_orm.InsertOrUpdateStructMulti(t.Context(), txn, trackedTable, []*TrackedEntity{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 0 {
		t.Errorf("expected 0 mutations, got %d", len(txn.mutations))
	}
}

func TestInsertOrUpdateStructMulti_MixedInsertAndUpdate(t *testing.T) {
	txn := &mockTxn{}
	newEntity := &TrackedEntity{TrackID: "id1", Name: "new"}
	existingEntity := &TrackedEntity{
		TrackID:  "id2",
		Name:     "updated",
		Original: &TrackedEntity{TrackID: "id2", Name: "old"},
	}
	entities := []*TrackedEntity{newEntity, existingEntity}
	if err := lure_orm.InsertOrUpdateStructMulti(t.Context(), txn, trackedTable, entities); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(txn.mutations))
	}
	// Insert path: CreatedAt set
	if newEntity.CreatedAt.IsZero() {
		t.Error("new entity: expected CreatedAt set")
	}
	// Update path: UpdatedAt set
	if existingEntity.UpdatedAt.IsZero() {
		t.Error("existing entity: expected UpdatedAt set")
	}
}

func TestInsertOrUpdateStructMulti_AllInserts(t *testing.T) {
	txn := &mockTxn{}
	entities := []*TrackedEntity{
		{TrackID: "id1", Name: "one"},
		{TrackID: "id2", Name: "two"},
	}
	if err := lure_orm.InsertOrUpdateStructMulti(t.Context(), txn, trackedTable, entities); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(txn.mutations))
	}
}

func TestInsertOrUpdateStructMulti_AllUpdates(t *testing.T) {
	txn := &mockTxn{}
	entities := []*TrackedEntity{
		{TrackID: "id1", Name: "new1", Original: &TrackedEntity{TrackID: "id1", Name: "old1"}},
		{TrackID: "id2", Name: "new2", Original: &TrackedEntity{TrackID: "id2", Name: "old2"}},
	}
	if err := lure_orm.InsertOrUpdateStructMulti(t.Context(), txn, trackedTable, entities); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(txn.mutations))
	}
}

// ============================================================
// DeleteMulti
// ============================================================

func TestDeleteMulti_EmptySlice(t *testing.T) {
	txn := &mockTxn{}
	if err := lure_orm.DeleteMulti(t.Context(), txn, trackedTable, []spanner.Key{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 0 {
		t.Errorf("expected 0 mutations, got %d", len(txn.mutations))
	}
}

func TestDeleteMulti_MultipleKeys(t *testing.T) {
	txn := &mockTxn{}
	keys := []spanner.Key{
		spanner.Key{"id1"},
		spanner.Key{"id2"},
		spanner.Key{"id3"},
	}
	if err := lure_orm.DeleteMulti(t.Context(), txn, trackedTable, keys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 3 {
		t.Errorf("expected 3 mutations, got %d", len(txn.mutations))
	}
}

func TestDeleteMulti_SingleKey(t *testing.T) {
	txn := &mockTxn{}
	if err := lure_orm.DeleteMulti(t.Context(), txn, trackedTable, []spanner.Key{spanner.Key{"id1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txn.mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(txn.mutations))
	}
}
