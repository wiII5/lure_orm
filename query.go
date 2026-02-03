package lure_orm

import (
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

// Query represents a SQL SELECT query builder.
type Query struct {
	columns    string
	table      string
	conditions []condition
	orderBy    string
	limit      int64
	offset     int64
	forceIndex string
	paramIndex int
}

type condition struct {
	sql  string
	args map[string]interface{}
}

// Select starts building a query with the given columns.
func Select(columns string) *Query {
	return &Query{columns: columns}
}

// From sets the table name.
func (q *Query) From(table string) *Query {
	q.table = table
	return q
}

// Where adds a raw WHERE condition with named parameters.
func (q *Query) Where(cond string, args ...interface{}) *Query {
	params := make(map[string]interface{})
	// Replace positional ? with named params
	replaced := cond
	for _, arg := range args {
		paramName := q.nextParam()
		replaced = strings.Replace(replaced, "?", "@"+paramName, 1)
		params[paramName] = arg
	}
	q.conditions = append(q.conditions, condition{sql: replaced, args: params})
	return q
}

// Eq adds a column = value condition.
func (q *Query) Eq(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s = @%s", column, paramName),
		args: map[string]interface{}{paramName: value},
	})
	return q
}

// NotEq adds a column != value condition.
func (q *Query) NotEq(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s != @%s", column, paramName),
		args: map[string]interface{}{paramName: value},
	})
	return q
}

// In adds a column IN (values) condition.
func (q *Query) In(column string, values interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s IN UNNEST(@%s)", column, paramName),
		args: map[string]interface{}{paramName: values},
	})
	return q
}

// IsNull adds a column IS NULL condition.
func (q *Query) IsNull(column string) *Query {
	q.conditions = append(q.conditions, condition{
		sql: fmt.Sprintf("%s IS NULL", column),
	})
	return q
}

// IsNotNull adds a column IS NOT NULL condition.
func (q *Query) IsNotNull(column string) *Query {
	q.conditions = append(q.conditions, condition{
		sql: fmt.Sprintf("%s IS NOT NULL", column),
	})
	return q
}

// Gt adds a column > value condition.
func (q *Query) Gt(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s > @%s", column, paramName),
		args: map[string]interface{}{paramName: value},
	})
	return q
}

// Gte adds a column >= value condition.
func (q *Query) Gte(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s >= @%s", column, paramName),
		args: map[string]interface{}{paramName: value},
	})
	return q
}

// Lt adds a column < value condition.
func (q *Query) Lt(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s < @%s", column, paramName),
		args: map[string]interface{}{paramName: value},
	})
	return q
}

// Lte adds a column <= value condition.
func (q *Query) Lte(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s <= @%s", column, paramName),
		args: map[string]interface{}{paramName: value},
	})
	return q
}

// Like adds a column LIKE pattern condition.
func (q *Query) Like(column string, pattern string) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:  fmt.Sprintf("%s LIKE @%s", column, paramName),
		args: map[string]interface{}{paramName: pattern},
	})
	return q
}

// Limit sets the LIMIT clause.
func (q *Query) Limit(n int64) *Query {
	q.limit = n
	return q
}

// Offset sets the OFFSET clause.
func (q *Query) Offset(n int64) *Query {
	q.offset = n
	return q
}

// OrderBy sets the ORDER BY clause.
func (q *Query) OrderBy(order string) *Query {
	q.orderBy = order
	return q
}

// ForceIndex adds the FORCE_INDEX hint to the table.
func (q *Query) ForceIndex(index string) *Query {
	q.forceIndex = index
	return q
}

// ToStatement builds the spanner.Statement.
func (q *Query) ToStatement() (spanner.Statement, error) {
	if q.table == "" {
		return spanner.Statement{}, fmt.Errorf("lure_orm: table name is required")
	}
	if q.columns == "" {
		return spanner.Statement{}, fmt.Errorf("lure_orm: columns are required")
	}

	var sb strings.Builder
	params := make(map[string]interface{})

	sb.WriteString("SELECT ")
	sb.WriteString(q.columns)
	sb.WriteString(" FROM ")
	sb.WriteString(q.table)

	if q.forceIndex != "" {
		sb.WriteString(fmt.Sprintf("@{FORCE_INDEX=%s}", q.forceIndex))
	}

	if len(q.conditions) > 0 {
		sb.WriteString(" WHERE ")
		for i, c := range q.conditions {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(c.sql)
			for k, v := range c.args {
				params[k] = v
			}
		}
	}

	if q.orderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(q.orderBy)
	}

	if q.limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", q.limit))
	}

	if q.offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", q.offset))
	}

	return spanner.Statement{
		SQL:    sb.String(),
		Params: params,
	}, nil
}

// ToCountStatement builds a COUNT(*) statement from this query.
func (q *Query) ToCountStatement() (spanner.Statement, error) {
	if q.table == "" {
		return spanner.Statement{}, fmt.Errorf("lure_orm: table name is required")
	}

	var sb strings.Builder
	params := make(map[string]interface{})

	sb.WriteString("SELECT COUNT(*) FROM ")
	sb.WriteString(q.table)

	if q.forceIndex != "" {
		sb.WriteString(fmt.Sprintf("@{FORCE_INDEX=%s}", q.forceIndex))
	}

	if len(q.conditions) > 0 {
		sb.WriteString(" WHERE ")
		for i, c := range q.conditions {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(c.sql)
			for k, v := range c.args {
				params[k] = v
			}
		}
	}

	return spanner.Statement{
		SQL:    sb.String(),
		Params: params,
	}, nil
}

func (q *Query) nextParam() string {
	q.paramIndex++
	return fmt.Sprintf("p%d", q.paramIndex)
}
