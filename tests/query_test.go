package tests

import (
	"strings"
	"testing"

	"github.bitech.jp/lure_orm"
)

func TestSelectBasic(t *testing.T) {
	q := lure_orm.Select("*").From("Users")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM Users"
	if stmt.SQL != expected {
		t.Errorf("expected %q, got %q", expected, stmt.SQL)
	}
}

func TestSelectWithColumns(t *testing.T) {
	q := lure_orm.Select("UserId, Email, Name").From("Users")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "SELECT UserId, Email, Name FROM Users") {
		t.Errorf("unexpected SQL: %s", stmt.SQL)
	}
}

func TestSelectWithEq(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Eq("Status", "active")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "WHERE Status = @p1") {
		t.Errorf("unexpected SQL: %s", stmt.SQL)
	}
	if stmt.Params["p1"] != "active" {
		t.Errorf("expected param p1 to be 'active', got %v", stmt.Params["p1"])
	}
}

func TestSelectWithMultipleConditions(t *testing.T) {
	q := lure_orm.Select("*").
		From("Users").
		Eq("Status", "active").
		Gt("Age", 18)
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Status = @p1") {
		t.Errorf("missing Status condition: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "Age > @p2") {
		t.Errorf("missing Age condition: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, " AND ") {
		t.Errorf("missing AND between conditions: %s", stmt.SQL)
	}
}

func TestSelectWithIn(t *testing.T) {
	q := lure_orm.Select("*").From("Users").In("UserId", []int64{1, 2, 3})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "UserId IN UNNEST(@p1)") {
		t.Errorf("unexpected SQL: %s", stmt.SQL)
	}
}

func TestSelectWithOrderByAndLimit(t *testing.T) {
	q := lure_orm.Select("*").
		From("Users").
		OrderBy("CreateTime DESC").
		Limit(10).
		Offset(20)
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "ORDER BY CreateTime DESC") {
		t.Errorf("missing ORDER BY: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "LIMIT 10") {
		t.Errorf("missing LIMIT: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, "OFFSET 20") {
		t.Errorf("missing OFFSET: %s", stmt.SQL)
	}
}

func TestSelectWithForceIndex(t *testing.T) {
	q := lure_orm.Select("*").From("Users").ForceIndex("UsersByEmail")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "@{FORCE_INDEX=UsersByEmail}") {
		t.Errorf("missing FORCE_INDEX hint: %s", stmt.SQL)
	}
}

func TestSelectWithLike(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Like("Email", "%@example.com")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Email LIKE @p1") {
		t.Errorf("unexpected SQL: %s", stmt.SQL)
	}
}

func TestSelectWithRawWhere(t *testing.T) {
	q := lure_orm.Select("*").From("Users").WhereRaw("Age >= ? AND Age <= ?", 18, 65)
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Age >= @p1 AND Age <= @p2") {
		t.Errorf("unexpected SQL: %s", stmt.SQL)
	}
	if stmt.Params["p1"] != 18 {
		t.Errorf("expected p1=18, got %v", stmt.Params["p1"])
	}
	if stmt.Params["p2"] != 65 {
		t.Errorf("expected p2=65, got %v", stmt.Params["p2"])
	}
}

func TestCountStatement(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Eq("Status", "active")
	stmt, err := q.ToCountStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(stmt.SQL, "SELECT COUNT(*) FROM Users") {
		t.Errorf("expected COUNT query, got: %s", stmt.SQL)
	}
}

func TestQueryWithoutTable(t *testing.T) {
	q := lure_orm.Select("*")
	_, err := q.ToStatement()
	if err == nil {
		t.Error("expected error for missing table name")
	}
}

func TestQueryWithoutColumns(t *testing.T) {
	q := &lure_orm.Query{}
	q.From("Users")
	_, err := q.ToStatement()
	if err == nil {
		t.Error("expected error for missing columns")
	}
}

func TestOrConditions(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Eq("Status", "active").OrEq("Status", "pending")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Status = @p1 OR Status = @p2") {
		t.Errorf("expected OR condition, got: %s", stmt.SQL)
	}
}

func TestMixedAndOrConditions(t *testing.T) {
	q := lure_orm.Select("*").From("Users").
		Eq("Type", "user").
		Eq("Status", "active").
		OrEq("Status", "pending")
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be: Type = @p1 AND Status = @p2 OR Status = @p3
	if !strings.Contains(stmt.SQL, "Type = @p1 AND Status = @p2 OR Status = @p3") {
		t.Errorf("expected mixed AND/OR, got: %s", stmt.SQL)
	}
}

func TestWhereGroup(t *testing.T) {
	q := lure_orm.Select("*").From("Users").
		Eq("Type", "user").
		WhereGroup(func(sub *lure_orm.Query) {
			sub.Eq("Status", "active").OrEq("Status", "pending")
		})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be: Type = @p1 AND (Status = @p2 OR Status = @p3)
	if !strings.Contains(stmt.SQL, "Type = @p1 AND (Status = @p2 OR Status = @p3)") {
		t.Errorf("expected grouped condition, got: %s", stmt.SQL)
	}
}

func TestOrWhereGroup(t *testing.T) {
	q := lure_orm.Select("*").From("Users").
		WhereGroup(func(sub *lure_orm.Query) {
			sub.Eq("Type", "admin").Eq("Status", "active")
		}).
		OrWhereGroup(func(sub *lure_orm.Query) {
			sub.Eq("Type", "user").Eq("Status", "active")
		})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be: (Type = @p1 AND Status = @p2) OR (Type = @p3 AND Status = @p4)
	if !strings.Contains(stmt.SQL, "(Type = @p1 AND Status = @p2) OR (Type = @p3 AND Status = @p4)") {
		t.Errorf("expected OR grouped conditions, got: %s", stmt.SQL)
	}
}

func TestOrIn(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Eq("Type", "user").OrIn("Status", []string{"active", "pending"})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Type = @p1 OR Status IN UNNEST(@p2)") {
		t.Errorf("expected OR IN, got: %s", stmt.SQL)
	}
}

func TestWhereWithCond(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Where(lure_orm.And{
		lure_orm.Eq{"PlayerId": "player1", "Email": "test@example.com"},
		lure_orm.GtOrEq{"CreatedAt": "2024-01-01"},
	})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain the conditions
	if !strings.Contains(stmt.SQL, "WHERE") {
		t.Errorf("expected WHERE clause, got: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, ">=") {
		t.Errorf("expected >= operator for GtOrEq, got: %s", stmt.SQL)
	}
}

func TestEqArr(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Where(lure_orm.EqArr{"Email": []string{"a@test.com", "b@test.com"}})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "Email IN UNNEST(@p1)") {
		t.Errorf("expected IN UNNEST for EqArr, got: %s", stmt.SQL)
	}
}

func TestWhereWithComplexCond(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Where(lure_orm.Or{
		lure_orm.And{
			lure_orm.Eq{"Type": "admin"},
			lure_orm.Eq{"Status": "active"},
		},
		lure_orm.And{
			lure_orm.Eq{"Type": "user"},
			lure_orm.Gte{"Level": 10},
		},
	})
	stmt, err := q.ToStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "WHERE") {
		t.Errorf("expected WHERE clause, got: %s", stmt.SQL)
	}
	if !strings.Contains(stmt.SQL, " OR ") {
		t.Errorf("expected OR in query, got: %s", stmt.SQL)
	}
}

func TestToStmt(t *testing.T) {
	q := lure_orm.Select("*").From("Users").Eq("Status", "active")
	stmt, err := q.ToStmt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stmt.SQL, "SELECT * FROM Users WHERE Status = @p1") {
		t.Errorf("unexpected SQL from ToStmt: %s", stmt.SQL)
	}
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		query    *lure_orm.Query
		expected string
	}{
		{
			name:     "NotEq",
			query:    lure_orm.Select("*").From("T").NotEq("Col", "val"),
			expected: "Col != @p1",
		},
		{
			name:     "Gt",
			query:    lure_orm.Select("*").From("T").Gt("Col", 10),
			expected: "Col > @p1",
		},
		{
			name:     "Gte",
			query:    lure_orm.Select("*").From("T").Gte("Col", 10),
			expected: "Col >= @p1",
		},
		{
			name:     "Lt",
			query:    lure_orm.Select("*").From("T").Lt("Col", 10),
			expected: "Col < @p1",
		},
		{
			name:     "Lte",
			query:    lure_orm.Select("*").From("T").Lte("Col", 10),
			expected: "Col <= @p1",
		},
		{
			name:     "IsNotNull",
			query:    lure_orm.Select("*").From("T").IsNotNull("Col"),
			expected: "Col IS NOT NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := tt.query.ToStatement()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stmt.SQL, tt.expected) {
				t.Errorf("expected SQL to contain %q, got: %s", tt.expected, stmt.SQL)
			}
		})
	}
}
