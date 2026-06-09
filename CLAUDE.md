# lure_orm — CLAUDE.md

## 概要

Cloud Spanner 用 Go ORM ライブラリ。lure_server から使用される。
Go Generics（1.23+）を活用し、型安全なクエリ・ミューテーション・タイムスタンプ自動管理を提供する。

**モジュール名:** `github.com/wiII5/lure_orm`

---

## ビルド・テスト

```bash
go test ./...                    # 全テスト実行
go test ./tests/...              # テストスイート
```

---

## ファイル構成

```
lure_orm/
├── lure_orm.go        # Row 型・QueryRow・ExecuteQuery・ReadRow
├── query.go           # Query ビルダー（Select/From/Where/OrderBy/Limit/Offset）
├── cond.go            # Cond インターフェース・全条件型定義
├── timestamps.go      # CreatedAt/UpdatedAt 自動管理
├── transaction.go     # ReadRunner / ReadWriteRunner インターフェース
├── iterator.go        # IterateAll / IterateOne / QueryAll / QueryOne 等
├── utils/
│   ├── utils.go       # TableName / ColumnNames / Contains 等ヘルパー
│   └── constants.go   # lure_orm タグ定数
└── logger/
    ├── config.go      # LogLevel / Config / Option
    └── log.go         # Logger 実装
```

---

## コアインターフェース

### ReadRunner / ReadWriteRunner

```go
type ReadRunner interface {
    Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
    Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) *spanner.RowIterator
    ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
}

type ReadWriteRunner interface {
    ReadRunner
    BufferWrite(ms []*spanner.Mutation) error
    Update(ctx context.Context, stmt spanner.Statement) (int64, error)
}
```

- `*spanner.ReadOnlyTransaction` → `ReadRunner` を満たす
- `*spanner.ReadWriteTransaction` → `ReadRunner` / `ReadWriteRunner` 両方を満たす

---

## クエリビルダー

### チェーンメソッド一覧

```go
lure_orm.Select(columns string) *Query          // カラム指定
  .From(table string) *Query                    // テーブル指定
  .Column(expr string) *Query                   // カラム追加
  .Columns(exprs ...string) *Query              // 複数カラム追加
  .Where(cond Cond) *Query                      // AND 条件追加
  .OrWhereCond(cond Cond) *Query                // OR 条件追加
  .WhereRaw(cond string, args ...any) *Query    // RAW AND 条件（? プレースホルダー）
  .WhereGroup(fn func(*Query)) *Query           // AND グループ: (A AND B)
  .OrWhereGroup(fn func(*Query)) *Query         // OR グループ: OR (A AND B)
  .OrderBy(order string) *Query                 // ORDER BY 句
  .Limit(n int64) *Query                        // LIMIT 句
  .Offset(n int64) *Query                       // OFFSET 句
  .ForceIndex(index string) *Query              // @{FORCE_INDEX=index} ヒント
  .ToStmt() (spanner.Statement, error)          // SELECT 文構築
  .ToCountStatement() (spanner.Statement, error) // SELECT COUNT(*) 文構築
```

### 実行関数（Generics）

```go
lure_orm.Find[T any](ctx, txn ReadRunner, q *Query) ([]*T, error)
lure_orm.FindOne[T any](ctx, txn ReadRunner, q *Query) (*T, error)
lure_orm.Count(ctx, txn ReadRunner, q *Query) (int64, error)
lure_orm.Exists(ctx, txn ReadRunner, q *Query) (bool, error)
```

---

## 条件型（Cond インターフェース実装）

```go
type Cond interface {
    build(paramIndex *int) (sql string, params map[string]interface{})
}
```

| 型 | Go 定義 | 生成される SQL |
|----|---------|---------------|
| `Eq` | `type Eq map[string]interface{}` | `col = @p1`（複数キー: AND 結合） |
| `NotEq` | `type NotEq map[string]interface{}` | `col != @p1` |
| `In` | `type In map[string]interface{}` | `col IN UNNEST(@p1)` |
| `NotIn` | `type NotIn map[string]interface{}` | `col NOT IN UNNEST(@p1)` |
| `Gt` | `type Gt map[string]interface{}` | `col > @p1` |
| `Gte` | `type Gte map[string]interface{}` | `col >= @p1` |
| `Lt` | `type Lt map[string]interface{}` | `col < @p1` |
| `Lte` | `type Lte map[string]interface{}` | `col <= @p1` |
| `Like` | `type Like map[string]string` | `col LIKE @p1` |
| `IsNull` | `type IsNull []string` | `col IS NULL`（複数: AND 結合） |
| `IsNotNull` | `type IsNotNull []string` | `col IS NOT NULL` |
| `And` | `type And []Cond` | 子条件を AND で結合（括弧付き） |
| `Or` | `type Or []Cond` | 子条件を OR で結合（括弧付き） |
| `Not` | `type Not struct{ Cond Cond }` | `NOT (condition)` |
| `Raw` | `type Raw struct{ SQL string; Args []interface{} }` | SQL そのまま（`?` → `@p1`） |

**重要:** `nil` スライスの `Or{}` は `build()` が `"", nil` を返すため WHERE 句なし（全件）になる。
条件が必要な場合のみ OR に要素を追加するパターンで「全件 or 絞り込み」を1メソッドに統合できる。

```go
// ✅ 全件 or キーワード絞り込みを統合
var where lure_orm.Or
if keyword != "" {
    kw := "%" + strings.ToLower(keyword) + "%"
    where = lure_orm.Or{
        lure_orm.Raw{SQL: "LOWER(Name) LIKE ?", Args: []interface{}{kw}},
    }
}
stmt, _ := lure_orm.Select(cols).From(table).Where(where).ToStmt()
```

---

## ミューテーション関数

### Struct ベース（推奨）

```go
// 単一操作
lure_orm.InsertStruct(ctx, txn ReadWriteRunner, table string, v interface{}) error
lure_orm.UpdateStruct(ctx, txn ReadWriteRunner, table string, v interface{}) error
lure_orm.InsertOrUpdateStruct(ctx, txn ReadWriteRunner, table string, v interface{}) error

// 一括操作（Generics）
lure_orm.InsertStructMulti[T any](ctx, txn ReadWriteRunner, table string, items []*T) error
lure_orm.UpdateStructMulti[T any](ctx, txn ReadWriteRunner, table string, items []*T) error
lure_orm.InsertOrUpdateStructMulti[T any](ctx, txn ReadWriteRunner, table string, items []*T) error
```

### プリミティブ操作

```go
lure_orm.Insert(ctx, txn, table string, columns []string, values []interface{}) error
lure_orm.Update(ctx, txn, table string, columns []string, values []interface{}, key spanner.Key) error
lure_orm.Delete(ctx, txn, table string, key spanner.Key) error
lure_orm.DeleteMulti(ctx, txn, table string, keys []spanner.Key) error
lure_orm.ExecUpdate(ctx, txn ReadWriteRunner, stmt spanner.Statement) (int64, error)
```

---

## イテレーター・クエリ実行

```go
// クエリ実行
lure_orm.QueryAll[T any](ctx, txn ReadRunner, stmt spanner.Statement) ([]*T, error)
lure_orm.QueryOne[T any](ctx, txn ReadRunner, stmt spanner.Statement) (*T, error)
lure_orm.QueryCount(ctx, txn ReadRunner, stmt spanner.Statement) (int64, error)
lure_orm.QueryExists(ctx, txn ReadRunner, stmt spanner.Statement) (bool, error)

// イテレーター変換
lure_orm.IterateAll[T any](iter *spanner.RowIterator) ([]*T, error)
lure_orm.IterateOne[T any](iter *spanner.RowIterator) (*T, error)

// 手動スキャン（ARRAY<STRUCT> 等の positional scan に使用）
lure_orm.QueryRow(ctx, txn ReadRunner, stmt spanner.Statement) *Row
lure_orm.ExecuteQuery(ctx, txn ReadRunner, stmt spanner.Statement) *spanner.RowIterator
lure_orm.IterateRows(ctx, txn ReadRunner, stmt spanner.Statement, fn func(*spanner.Row) error) error
lure_orm.ReadRow[T any](ctx, txn ReadRunner, table string, key spanner.Key, columns []string) (*T, error)

// Row 型メソッド
row.Scan(ptrs ...interface{}) error  // positional scan
row.ToStruct(p interface{}) error    // struct へマップ（spanner タグ使用）
row.Err() error
```

**重要:** `QueryOne` / `IterateOne` は行なし時に `nil, nil` を返す。
`QueryRow().Scan()` / `QueryRow().ToStruct()` は行なし時に `lure_orm.ErrNoRows` を返す。

```go
var ErrNoRows = errors.New("lure_orm: no rows in result set")
```

ARRAY\<STRUCT\> を含むクエリには `QueryRow().Scan()` を使う（`QueryOne` は `row.ToStruct` を内部使用するため ARRAY\<STRUCT\> 非対応）。

---

## タイムスタンプ自動管理

`CreatedAt time.Time` / `UpdatedAt time.Time` フィールドをリフレクションで自動設定する。

| 操作 | CreatedAt | UpdatedAt |
|------|-----------|-----------|
| `InsertStruct` | `time.Now()` | `time.Now()` |
| `UpdateStruct` | `Original.CreatedAt` を復元 | `time.Now()` |
| `InsertOrUpdateStruct`（Original なし） | `time.Now()` | `time.Now()` |
| `InsertOrUpdateStruct`（Original あり） | `Original.CreatedAt` を復元 | `time.Now()` |
| Multiバリアント | 上記と同様 per-item | 同上 |

**制約:**
- フィールドは必ず `time.Time` 型（`spanner.NullTime` は不可）
- `Original` フィールドには必ず `spanner:"-"` タグを付ける

---

## struct タグ規約

```go
type MyEntity struct {
    MyId      string    `spanner:"MyId" lure_orm:"primary"`  // PK
    Name      string    `spanner:"Name"`
    CreatedAt time.Time `spanner:"CreatedAt"`                 // lure_orm 自動管理
    UpdatedAt time.Time `spanner:"UpdatedAt"`                 // lure_orm 自動管理
    Original  *MyEntity `spanner:"-"`                         // Spanner スキャン対象外
}
```

| タグ | 値 | 用途 |
|------|-----|------|
| `spanner` | カラム名 | Spanner カラムへのマッピング |
| `spanner` | `"-"` | Spanner スキャン対象外（Original に必須） |
| `lure_orm` | `"primary"` | PK フィールドのマーク |

### EntityWithPK インターフェース

差分更新（UpdateStruct）を有効にするために実装が必要：

```go
func (e *MyEntity) SpannerPrimaryKeyColumns() []string {
    return []string{"MyId"}
}
```

---

## ロガー

```go
type LogLevel int
const (
    LogLevelNone  LogLevel = iota  // 0: ログなし
    LogLevelRead                    // 1: 読み取りのみ
    LogLevelWrite                   // 2: 書き込みのみ
    LogLevelAll                     // 3: 全ログ
)

log := logger.New(
    logger.WithLogLevel(logger.LogLevelAll),
    logger.WithFields(map[string]any{"service": "lure_server"}),
)
log.Read(ctx, "SELECT ...")
log.Write(ctx, "INSERT ...")
log.Error(ctx, err, "failed")
```

---

## 禁止事項

- `CreatedAt` / `UpdatedAt` をアプリ側でセットしない（lure_orm が自動管理）
- `UpdateStruct` 使用時に `Original` をセットせずに呼ぶと `CreatedAt` が失われる
- `QueryOne` を ARRAY\<STRUCT\> クエリに使用しない（内部で `row.ToStruct` を使うため非対応）
- ループ内で `InsertStruct` / `UpdateStruct` を個別に呼ぶ N+1 書き込みを行わない（Multi バリアントを使う）
