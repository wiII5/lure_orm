package lure_orm

import (
	"fmt"
	"strings"
)

// Cond represents a condition that can be used in WHERE clauses.
type Cond interface {
	build(paramIndex *int) (sql string, params map[string]interface{})
}

// Eq represents equality conditions: column = value
// Usage: Eq{"column": value} or Eq{"col1": val1, "col2": val2}
type Eq map[string]interface{}

func (e Eq) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s = @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// EqArr is deprecated: use In instead.
// EqArr represents equality conditions with array values: column IN UNNEST(values)
type EqArr = In

// NotEq represents not-equality conditions: column != value
type NotEq map[string]interface{}

func (e NotEq) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s != @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// In represents IN conditions: column IN UNNEST(values)
// Usage: In{"column": []string{"a", "b", "c"}}
type In map[string]interface{}

func (e In) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s IN UNNEST(@%s)", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// NotIn represents NOT IN conditions: column NOT IN UNNEST(values)
type NotIn map[string]interface{}

func (e NotIn) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s NOT IN UNNEST(@%s)", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// Gt represents greater-than conditions: column > value
type Gt map[string]interface{}

func (e Gt) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s > @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// Gte represents greater-than-or-equal conditions: column >= value
type Gte map[string]interface{}

func (e Gte) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s >= @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// GtOrEq is an alias for Gte (greater-than-or-equal conditions): column >= value
type GtOrEq = Gte

// Lt represents less-than conditions: column < value
type Lt map[string]interface{}

func (e Lt) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s < @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// Lte represents less-than-or-equal conditions: column <= value
type Lte map[string]interface{}

func (e Lte) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s <= @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// LtOrEq is an alias for Lte (less-than-or-equal conditions): column <= value
type LtOrEq = Lte

// Like represents LIKE conditions: column LIKE pattern
type Like map[string]string

func (e Like) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for col, val := range e {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		parts = append(parts, fmt.Sprintf("%s LIKE @%s", col, paramName))
		params[paramName] = val
	}

	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// IsNull represents IS NULL conditions
// Usage: IsNull{"column"} or IsNull{"col1", "col2"}
type IsNull []string

func (e IsNull) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	for _, col := range e {
		parts = append(parts, fmt.Sprintf("%s IS NULL", col))
	}

	if len(parts) == 1 {
		return parts[0], nil
	}
	return "(" + strings.Join(parts, " AND ") + ")", nil
}

// IsNotNull represents IS NOT NULL conditions
type IsNotNull []string

func (e IsNotNull) build(paramIndex *int) (string, map[string]interface{}) {
	if len(e) == 0 {
		return "", nil
	}

	var parts []string
	for _, col := range e {
		parts = append(parts, fmt.Sprintf("%s IS NOT NULL", col))
	}

	if len(parts) == 1 {
		return parts[0], nil
	}
	return "(" + strings.Join(parts, " AND ") + ")", nil
}

// And represents a group of conditions joined by AND
// Usage: And{Eq{"col1": val1}, Eq{"col2": val2}}
type And []Cond

func (a And) build(paramIndex *int) (string, map[string]interface{}) {
	if len(a) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for _, cond := range a {
		sql, p := cond.build(paramIndex)
		if sql != "" {
			parts = append(parts, sql)
			for k, v := range p {
				params[k] = v
			}
		}
	}

	if len(parts) == 0 {
		return "", nil
	}
	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " AND ") + ")", params
}

// Or represents a group of conditions joined by OR
// Usage: Or{Eq{"col1": val1}, Eq{"col2": val2}}
type Or []Cond

func (o Or) build(paramIndex *int) (string, map[string]interface{}) {
	if len(o) == 0 {
		return "", nil
	}

	var parts []string
	params := make(map[string]interface{})

	for _, cond := range o {
		sql, p := cond.build(paramIndex)
		if sql != "" {
			parts = append(parts, sql)
			for k, v := range p {
				params[k] = v
			}
		}
	}

	if len(parts) == 0 {
		return "", nil
	}
	if len(parts) == 1 {
		return parts[0], params
	}
	return "(" + strings.Join(parts, " OR ") + ")", params
}

// Not represents a negated condition: NOT (condition)
// Usage: Not{Eq{"col": val}} generates "NOT (col = @p1)"
type Not struct {
	Cond Cond
}

func (n Not) build(paramIndex *int) (string, map[string]interface{}) {
	if n.Cond == nil {
		return "", nil
	}

	sql, params := n.Cond.build(paramIndex)
	if sql == "" {
		return "", nil
	}

	return "NOT (" + sql + ")", params
}

// Raw represents a raw SQL condition with parameters
// Usage: Raw{"col > ? AND col < ?", val1, val2}
type Raw struct {
	SQL  string
	Args []interface{}
}

func (r Raw) build(paramIndex *int) (string, map[string]interface{}) {
	if r.SQL == "" {
		return "", nil
	}

	params := make(map[string]interface{})
	sql := r.SQL

	for _, arg := range r.Args {
		*paramIndex++
		paramName := fmt.Sprintf("p%d", *paramIndex)
		sql = strings.Replace(sql, "?", "@"+paramName, 1)
		params[paramName] = arg
	}

	return sql, params
}
