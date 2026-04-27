package tests

import (
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/wiII5/lure_orm/utils"
)

func TestTableName(t *testing.T) {
	name := utils.TableName[User]()
	if name != "User" {
		t.Errorf("expected 'User', got %q", name)
	}
}

func TestFindPrimaryKeyColumn(t *testing.T) {
	pk := utils.FindPrimaryKeyColumn[User]()
	if pk != "UserId" {
		t.Errorf("expected 'UserId', got %q", pk)
	}
}

func TestIsNullable(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"NullInt64", spanner.NullInt64{}, true},
		{"NullString", spanner.NullString{}, true},
		{"NullFloat64", spanner.NullFloat64{}, true},
		{"NullTime", spanner.NullTime{}, true},
		{"NullDate", spanner.NullDate{}, true},
		{"NullBool", spanner.NullBool{}, true},
		{"int64", int64(0), false},
		{"string", "", false},
		{"time.Time", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsNullable(tt.value)
			if result != tt.expected {
				t.Errorf("IsNullable(%T) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestIsTime(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"time.Time", time.Now(), true},
		{"*time.Time", new(time.Time), true},
		{"NullTime", spanner.NullTime{}, true},
		{"string", "", false},
		{"int64", int64(0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsTime(tt.value)
			if result != tt.expected {
				t.Errorf("IsTime(%T) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestIsZeroTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"zero time.Time", time.Time{}, true},
		{"non-zero time.Time", now, false},
		{"nil *time.Time", (*time.Time)(nil), true},
		{"zero *time.Time", &time.Time{}, true},
		{"non-zero *time.Time", &now, false},
		{"invalid NullTime", spanner.NullTime{Valid: false}, true},
		{"valid NullTime", spanner.NullTime{Valid: true, Time: now}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsZeroTime(tt.value)
			if result != tt.expected {
				t.Errorf("IsZeroTime(%v) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !utils.Contains(slice, "b") {
		t.Error("expected Contains to return true for 'b'")
	}
	if utils.Contains(slice, "d") {
		t.Error("expected Contains to return false for 'd'")
	}

	intSlice := []int{1, 2, 3}
	if !utils.Contains(intSlice, 2) {
		t.Error("expected Contains to return true for 2")
	}
}

func TestColumnNames(t *testing.T) {
	names := utils.ColumnNames[User]()
	if names == "" {
		t.Error("expected non-empty column names")
	}
	// Check that it contains expected column names
	if !containsSubstring(names, "UserId") {
		t.Errorf("expected column names to contain 'UserId', got: %s", names)
	}
	if !containsSubstring(names, "Email") {
		t.Errorf("expected column names to contain 'Email', got: %s", names)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
