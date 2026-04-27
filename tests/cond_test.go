package tests

import (
	"strings"
	"testing"

	lure_orm "github.com/wiII5/lure_orm"
)

func TestIsNull_SingleColumn(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.IsNull{"Col"})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Col IS NULL") {
		t.Errorf("expected 'Col IS NULL', got: %s", stmt.SQL)
	}
	if len(stmt.Params) != 0 {
		t.Errorf("expected no params for IS NULL, got %v", stmt.Params)
	}
}

func TestIsNull_MultipleColumns(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.IsNull{"Col1", "Col2"})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Col1 IS NULL") {
		t.Errorf("expected 'Col1 IS NULL', got: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "Col2 IS NULL") {
		t.Errorf("expected 'Col2 IS NULL', got: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, " AND ") {
		t.Errorf("expected AND between IS NULL conditions, got: %s", stmt.SQL)
	}
}

func TestIsNotNull_SingleColumn(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.IsNotNull{"Col"})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Col IS NOT NULL") {
		t.Errorf("expected 'Col IS NOT NULL', got: %s", stmt.SQL)
	}
}

func TestIsNotNull_MultipleColumns(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.IsNotNull{"Col1", "Col2"})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Col1 IS NOT NULL") {
		t.Errorf("expected 'Col1 IS NOT NULL', got: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "Col2 IS NOT NULL") {
		t.Errorf("expected 'Col2 IS NOT NULL', got: %s", stmt.SQL)
	}
}

func TestNot_NegatesCondition(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.Not{Cond: lure_orm.Eq{"Status": "active"}})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "NOT (Status = @p1)") {
		t.Errorf("expected 'NOT (Status = @p1)', got: %s", stmt.SQL)
	}
	if stmt.Params["p1"] != "active" {
		t.Errorf("expected param p1='active', got %v", stmt.Params["p1"])
	}
}

func TestNot_NilCond(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.Not{Cond: nil})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Nil condition should produce no WHERE clause
	if strings.Contains(stmt.SQL, "NOT") {
		t.Errorf("expected no NOT clause for nil cond, got: %s", stmt.SQL)
	}
}

func TestNot_WithInCondition(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.Not{
		Cond: lure_orm.In{"Status": []string{"banned", "suspended"}},
	})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "NOT (Status IN UNNEST(@p1))") {
		t.Errorf("expected 'NOT (Status IN UNNEST(@p1))', got: %s", stmt.SQL)
	}
}

func TestRaw_WithArgs(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.Raw{
		SQL:  "Score BETWEEN ? AND ?",
		Args: []interface{}{10, 100},
	})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Score BETWEEN @p1 AND @p2") {
		t.Errorf("expected 'Score BETWEEN @p1 AND @p2', got: %s", stmt.SQL)
	}
	if stmt.Params["p1"] != 10 {
		t.Errorf("expected p1=10, got %v", stmt.Params["p1"])
	}
	if stmt.Params["p2"] != 100 {
		t.Errorf("expected p2=100, got %v", stmt.Params["p2"])
	}
}

func TestRaw_NoArgs(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.Raw{SQL: "IsActive = TRUE"})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "IsActive = TRUE") {
		t.Errorf("expected 'IsActive = TRUE', got: %s", stmt.SQL)
	}
}

func TestRaw_EmptySQL(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.Raw{SQL: ""})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty Raw should produce no WHERE clause
	if strings.Contains(stmt.SQL, "WHERE") {
		t.Errorf("expected no WHERE for empty Raw, got: %s", stmt.SQL)
	}
}

func TestNotIn_SingleColumn(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.NotIn{"Status": []string{"banned", "deleted"}})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Status NOT IN UNNEST(@p1)") {
		t.Errorf("expected 'Status NOT IN UNNEST(@p1)', got: %s", stmt.SQL)
	}
}

func TestNotIn_MultipleColumns(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.NotIn{
		"Status": []string{"banned"},
		"Type":   []string{"temp"},
	})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "NOT IN UNNEST") {
		t.Errorf("expected NOT IN UNNEST, got: %s", stmt.SQL)
	}
}

func TestNot_CombinedWithAnd(t *testing.T) {
	q := lure_orm.Select("*").From("T").Where(lure_orm.And{
		lure_orm.Eq{"Type": "user"},
		lure_orm.Not{Cond: lure_orm.Eq{"Status": "banned"}},
	})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Type = @p1") {
		t.Errorf("expected Type condition, got: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "NOT (Status = @p2)") {
		t.Errorf("expected NOT condition, got: %s", stmt.SQL)
	}
}
