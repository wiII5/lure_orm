package lure_orm

import (
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

// Query represents a SQL SELECT query builder.
type Query struct {
	columns       string
	extraColumns  []string
	table         string
	conditions    []condition
	orderBy       string
	limit         int64
	offset        int64
	forceIndex    string
	paramIndex    int
}

type conditionType int

const (
	conditionAnd conditionType = iota
	conditionOr
)

type condition struct {
	sql      string
	args     map[string]interface{}
	condType conditionType
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

// Column adds an additional column expression to the SELECT clause.
// Useful for ARRAY subqueries like:
//
//	Column(fmt.Sprintf(`ARRAY(SELECT AS STRUCT %s FROM %s WHERE ...) AS Items`, columns, table))
func (q *Query) Column(expr string) *Query {
	q.extraColumns = append(q.extraColumns, expr)
	return q
}

// Columns adds multiple column expressions to the SELECT clause.
func (q *Query) Columns(exprs ...string) *Query {
	q.extraColumns = append(q.extraColumns, exprs...)
	return q
}

// Where adds a condition using a Cond object (AND).
// Usage: Where(lure_orm.And{lure_orm.Eq{"col": val}, lure_orm.GtOrEq{"date": now}})
func (q *Query) Where(cond Cond) *Query {
	sql, params := cond.build(&q.paramIndex)
	if sql != "" {
		q.conditions = append(q.conditions, condition{sql: sql, args: params, condType: conditionAnd})
	}
	return q
}

// OrWhereCond adds a condition using a Cond object (OR).
// Usage: OrWhereCond(lure_orm.Eq{"status": "active"})
func (q *Query) OrWhereCond(cond Cond) *Query {
	sql, params := cond.build(&q.paramIndex)
	if sql != "" {
		q.conditions = append(q.conditions, condition{sql: sql, args: params, condType: conditionOr})
	}
	return q
}

// WhereRaw adds a raw WHERE condition with named parameters (AND).
// Usage: WhereRaw("column = ?", value)
func (q *Query) WhereRaw(cond string, args ...interface{}) *Query {
	params := make(map[string]interface{})
	// Replace positional ? with named params
	replaced := cond
	for _, arg := range args {
		paramName := q.nextParam()
		replaced = strings.Replace(replaced, "?", "@"+paramName, 1)
		params[paramName] = arg
	}
	q.conditions = append(q.conditions, condition{sql: replaced, args: params, condType: conditionAnd})
	return q
}

// OrWhereRaw adds a raw WHERE condition with named parameters (OR).
func (q *Query) OrWhereRaw(cond string, args ...interface{}) *Query {
	params := make(map[string]interface{})
	replaced := cond
	for _, arg := range args {
		paramName := q.nextParam()
		replaced = strings.Replace(replaced, "?", "@"+paramName, 1)
		params[paramName] = arg
	}
	q.conditions = append(q.conditions, condition{sql: replaced, args: params, condType: conditionOr})
	return q
}

// Eq adds a column = value condition (AND).
func (q *Query) Eq(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s = @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionAnd,
	})
	return q
}

// OrEq adds a column = value condition (OR).
func (q *Query) OrEq(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s = @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionOr,
	})
	return q
}

// NotEq adds a column != value condition (AND).
func (q *Query) NotEq(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s != @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionAnd,
	})
	return q
}

// OrNotEq adds a column != value condition (OR).
func (q *Query) OrNotEq(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s != @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionOr,
	})
	return q
}

// In adds a column IN (values) condition (AND).
func (q *Query) In(column string, values interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s IN UNNEST(@%s)", column, paramName),
		args:     map[string]interface{}{paramName: values},
		condType: conditionAnd,
	})
	return q
}

// OrIn adds a column IN (values) condition (OR).
func (q *Query) OrIn(column string, values interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s IN UNNEST(@%s)", column, paramName),
		args:     map[string]interface{}{paramName: values},
		condType: conditionOr,
	})
	return q
}

// IsNull adds a column IS NULL condition (AND).
func (q *Query) IsNull(column string) *Query {
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s IS NULL", column),
		condType: conditionAnd,
	})
	return q
}

// OrIsNull adds a column IS NULL condition (OR).
func (q *Query) OrIsNull(column string) *Query {
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s IS NULL", column),
		condType: conditionOr,
	})
	return q
}

// IsNotNull adds a column IS NOT NULL condition (AND).
func (q *Query) IsNotNull(column string) *Query {
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s IS NOT NULL", column),
		condType: conditionAnd,
	})
	return q
}

// OrIsNotNull adds a column IS NOT NULL condition (OR).
func (q *Query) OrIsNotNull(column string) *Query {
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s IS NOT NULL", column),
		condType: conditionOr,
	})
	return q
}

// Gt adds a column > value condition (AND).
func (q *Query) Gt(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s > @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionAnd,
	})
	return q
}

// OrGt adds a column > value condition (OR).
func (q *Query) OrGt(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s > @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionOr,
	})
	return q
}

// Gte adds a column >= value condition (AND).
func (q *Query) Gte(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s >= @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionAnd,
	})
	return q
}

// OrGte adds a column >= value condition (OR).
func (q *Query) OrGte(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s >= @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionOr,
	})
	return q
}

// Lt adds a column < value condition (AND).
func (q *Query) Lt(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s < @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionAnd,
	})
	return q
}

// OrLt adds a column < value condition (OR).
func (q *Query) OrLt(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s < @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionOr,
	})
	return q
}

// Lte adds a column <= value condition (AND).
func (q *Query) Lte(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s <= @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionAnd,
	})
	return q
}

// OrLte adds a column <= value condition (OR).
func (q *Query) OrLte(column string, value interface{}) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s <= @%s", column, paramName),
		args:     map[string]interface{}{paramName: value},
		condType: conditionOr,
	})
	return q
}

// Like adds a column LIKE pattern condition (AND).
func (q *Query) Like(column string, pattern string) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s LIKE @%s", column, paramName),
		args:     map[string]interface{}{paramName: pattern},
		condType: conditionAnd,
	})
	return q
}

// OrLike adds a column LIKE pattern condition (OR).
func (q *Query) OrLike(column string, pattern string) *Query {
	paramName := q.nextParam()
	q.conditions = append(q.conditions, condition{
		sql:      fmt.Sprintf("%s LIKE @%s", column, paramName),
		args:     map[string]interface{}{paramName: pattern},
		condType: conditionOr,
	})
	return q
}

// WhereGroup adds a grouped condition with AND.
// Example: WhereGroup(func(q *Query) { q.Eq("a", 1).Eq("b", 2) }) generates "(a = @p1 AND b = @p2)"
func (q *Query) WhereGroup(fn func(*Query)) *Query {
	sub := &Query{paramIndex: q.paramIndex}
	fn(sub)
	q.paramIndex = sub.paramIndex

	if len(sub.conditions) > 0 {
		sql, args := sub.buildConditionGroup()
		q.conditions = append(q.conditions, condition{
			sql:      "(" + sql + ")",
			args:     args,
			condType: conditionAnd,
		})
	}
	return q
}

// OrWhereGroup adds a grouped condition with OR.
// Example: OrWhereGroup(func(q *Query) { q.Eq("a", 1).Eq("b", 2) }) generates "OR (a = @p1 AND b = @p2)"
func (q *Query) OrWhereGroup(fn func(*Query)) *Query {
	sub := &Query{paramIndex: q.paramIndex}
	fn(sub)
	q.paramIndex = sub.paramIndex

	if len(sub.conditions) > 0 {
		sql, args := sub.buildConditionGroup()
		q.conditions = append(q.conditions, condition{
			sql:      "(" + sql + ")",
			args:     args,
			condType: conditionOr,
		})
	}
	return q
}

func (q *Query) buildConditionGroup() (string, map[string]interface{}) {
	var sb strings.Builder
	params := make(map[string]interface{})

	for i, c := range q.conditions {
		if i > 0 {
			if c.condType == conditionOr {
				sb.WriteString(" OR ")
			} else {
				sb.WriteString(" AND ")
			}
		}
		sb.WriteString(c.sql)
		for k, v := range c.args {
			params[k] = v
		}
	}

	return sb.String(), params
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

// ToStmt builds the spanner.Statement (alias for ToStatement).
func (q *Query) ToStmt() (spanner.Statement, error) {
	return q.ToStatement()
}

// ToStatement builds the spanner.Statement.
func (q *Query) ToStatement() (spanner.Statement, error) {
	if q.table == "" {
		return spanner.Statement{}, fmt.Errorf("lure_orm: table name is required")
	}
	if q.columns == "" && len(q.extraColumns) == 0 {
		return spanner.Statement{}, fmt.Errorf("lure_orm: columns are required")
	}

	var sb strings.Builder
	params := make(map[string]interface{})

	sb.WriteString("SELECT ")
	sb.WriteString(q.columns)
	// Append extra columns
	for _, col := range q.extraColumns {
		if q.columns != "" || len(q.extraColumns) > 1 {
			sb.WriteString(", ")
		}
		sb.WriteString(col)
	}
	sb.WriteString(" FROM ")
	sb.WriteString(q.table)

	if q.forceIndex != "" {
		sb.WriteString(fmt.Sprintf("@{FORCE_INDEX=%s}", q.forceIndex))
	}

	if len(q.conditions) > 0 {
		sb.WriteString(" WHERE ")
		for i, c := range q.conditions {
			if i > 0 {
				if c.condType == conditionOr {
					sb.WriteString(" OR ")
				} else {
					sb.WriteString(" AND ")
				}
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
				if c.condType == conditionOr {
					sb.WriteString(" OR ")
				} else {
					sb.WriteString(" AND ")
				}
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
