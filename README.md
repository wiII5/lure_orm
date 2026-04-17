# lure_orm

lure_orm is a Go generics-based ORM for Google Cloud Spanner, designed for lure_server.

## Overview

Unlike traditional ORMs that use pointer operations (`Model(&result)` + `Find` returning only `error`), lure_orm uses Go generics to return results directly.

### Features

- **Query Builder**: Fluent API for building SELECT queries
- **Generic Functions**: `Find[T]`, `FindOne[T]`, `Count`, `Exists`
- **Write Operations**: `Insert`, `Update`, `Delete` using Spanner mutations
- **Logging**: Configurable query logging with read/write filtering
- **Type Safe**: Leverages Go generics for compile-time type safety

### Supported Data Types

- STRING / spanner.NullString
- INT64 / spanner.NullInt64
- FLOAT64 / spanner.NullFloat64
- BOOL
- DATE / civil.Date / spanner.NullDate
- TIMESTAMP / time.Time / spanner.NullTime
- ARRAY types

## Installation

```go
import "github.bitech.jp/lure_orm"
```

## Usage

### SELECT Queries

```go
// Find all matching rows
q := lure_orm.Select("*").
    From("Users").
    Eq("Status", "active").
    OrderBy("CreatedAt DESC").
    Limit(10)
users, err := lure_orm.Find[User](ctx, txn, q)

// Find one row
q := lure_orm.Select("*").
    From("Users").
    Eq("Email", email).
    Limit(1)
user, err := lure_orm.FindOne[User](ctx, txn, q)

// Count rows
q := lure_orm.Select("*").From("Users").Eq("Status", "active")
count, err := lure_orm.Count(ctx, txn, q)

// Check existence
exists, err := lure_orm.Exists(ctx, txn, q)
```

### Query Builder Methods

```go
lure_orm.Select(columns).           // SELECT columns
    From(table).                    // FROM table
    Eq(column, value).              // column = value
    NotEq(column, value).           // column != value
    In(column, values).             // column IN UNNEST(values)
    IsNull(column).                 // column IS NULL
    IsNotNull(column).              // column IS NOT NULL
    Gt(column, value).              // column > value
    Gte(column, value).             // column >= value
    Lt(column, value).              // column < value
    Lte(column, value).             // column <= value
    Like(column, pattern).          // column LIKE pattern
    Where(cond, args...).           // raw WHERE condition
    OrderBy(order).                 // ORDER BY order
    Limit(n).                       // LIMIT n
    Offset(n).                      // OFFSET n
    ForceIndex(index)               // @{FORCE_INDEX=index}
```

### Write Operations

```go
// Insert
err := lure_orm.Insert(ctx, txn, "Users",
    []string{"UserId", "Name", "Email"},
    []interface{}{userId, name, email})

// Update
err := lure_orm.Update(ctx, txn, "Users",
    []string{"UserId", "Name", "UpdatedAt"},
    []interface{}{userId, newName, time.Now()},
    spanner.Key{userId})

// Delete
err := lure_orm.Delete(ctx, txn, "Users", spanner.Key{userId})
```

### With Logging

```go
import "github.bitech.jp/lure_orm/logger"

log := logger.New(
    logger.WithLogLevel(logger.LogLevelAll),
    logger.WithFields(map[string]any{"service": "my-app"}),
)

q := lure_orm.Select("*").From("Users").Eq("Id", id)
stmt, _ := q.ToStatement()
log.Read(ctx, stmt.SQL)

users, err := lure_orm.Find[User](ctx, txn, q)
```

## Comparison with Traditional ORM

### Before (pointer-based)
```go
var account *AdminAccount
err := orm.Model(&account).
    Where("Email = ?", email).
    First(ctx, txn)
// account is populated via pointer
```

### After (lure_orm with generics)
```go
q := lure_orm.Select(AdminAccountColumnNames).
    From(AdminAccountTableName).
    Eq(AdminAccountColumn_Email, email).
    Limit(1)
account, err := lure_orm.FindOne[AdminAccount](ctx, txn, q)
// account is returned directly
```

## Testing

```bash
go test -v ./tests/...
```

## License

MIT License
