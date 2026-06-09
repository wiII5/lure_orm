# lure_orm 技術仕様書

> Go Generics ベース Cloud Spanner ORM ライブラリ  
> モジュール: `github.com/wiII5/lure_orm`  
> 最終更新: 2026-06-02

---

## 目次

1. [概要](#1-概要)
2. [パッケージ構成](#2-パッケージ構成)
3. [コアインターフェース](#3-コアインターフェース)
4. [クエリビルダー（query.go）](#4-クエリビルダーquerygo)
5. [条件型（cond.go）](#5-条件型condgo)
6. [クエリ実行関数（lure_orm.go）](#6-クエリ実行関数lure_ormgo)
7. [ミューテーション関数（lure_orm.go）](#7-ミューテーション関数lure_ormgo)
8. [タイムスタンプ管理（timestamps.go）](#8-タイムスタンプ管理timestampsgo)
9. [差分 UPDATE（Diff-UPDATE）](#9-差分-updatediff-update)
10. [イテレーター（iterator.go）](#10-イテレーターiteratorgo)
11. [ユーティリティ（utils/utils.go）](#11-ユーティリティutilsutilsgo)
12. [ロガー（logger/）](#12-ロガーlogger)
13. [struct タグ規則](#13-struct-タグ規則)
14. [ベストプラクティス](#14-ベストプラクティス)
15. [既知の制限事項](#15-既知の制限事項)

---

## 1. 概要

lure_orm は Cloud Spanner 向けの軽量 ORM ライブラリ。主な機能：

- **フルエント API によるクエリビルダー**（SELECT / WHERE / ORDER / LIMIT / OFFSET）
- **Go Generics 対応**の型安全なクエリ実行関数
- **自動タイムスタンプ管理**（`CreatedAt` / `UpdatedAt`）
- **差分 UPDATE**（変更フィールドのみを更新する Partial UPDATE）
- **バッチ操作**（InsertStructMulti / UpdateStructMulti / InsertOrUpdateStructMulti）

---

## 2. パッケージ構成

```
lure_orm/
├── lure_orm.go       # クエリ実行・ミューテーション関数
├── query.go          # クエリビルダー（フルエント API）
├── cond.go           # 条件型（Eq, In, Gt, And, Or 等）
├── timestamps.go     # タイムスタンプ自動管理
├── transaction.go    # ReadRunner / ReadWriteRunner インターフェース
├── iterator.go       # イテレーターヘルパー
├── utils/
│   ├── utils.go      # リフレクションヘルパー（TableName, ColumnNames 等）
│   └── constants.go  # タグ定数（TagSpanner, TagLureORM 等）
└── logger/
    ├── config.go     # ロガー設定・オプション
    └── log.go        # ロガー実装（Read/Write/Error）
```

---

## 3. コアインターフェース

### ReadRunner

```go
type ReadRunner interface {
    Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
    Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) *spanner.RowIterator
    ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
}
```

実装: `*spanner.ReadOnlyTransaction`, `*spanner.ReadWriteTransaction`

### ReadWriteRunner

```go
type ReadWriteRunner interface {
    ReadRunner
    BufferWrite(ms []*spanner.Mutation) error
    Update(ctx context.Context, stmt spanner.Statement) (int64, error)
}
```

実装: `*spanner.ReadWriteTransaction`

### EntityWithPK（差分 UPDATE 用）

```go
type EntityWithPK interface {
    SpannerPrimaryKeyColumns() []string
}
```

このインターフェースを実装した Entity に `Original` フィールドが設定されている場合、`UpdateStruct()` は **変更フィールドのみ** を UPDATE する。

---

## 4. クエリビルダー（query.go）

### 基本構文

```go
q := lure_orm.Select("Col1, Col2, Col3").
    From("TableName").
    Where(lure_orm.Eq{"Status": "active"}).
    OrderBy("CreatedAt DESC").
    Limit(10).
    Offset(0)

stmt, err := q.ToStmt()
```

### 選択・テーブル指定

| メソッド | 説明 |
|---------|------|
| `Select(columns string) *Query` | SELECT 対象カラムを指定（必須） |
| `From(table string) *Query` | テーブル名を指定（必須） |
| `Column(expr string) *Query` | 単一カラム式を追加（例: ARRAY サブクエリ） |
| `Columns(exprs ...string) *Query` | 複数カラム式を追加 |

### WHERE 条件（Cond 型推奨）

| メソッド | 説明 |
|---------|------|
| `Where(cond Cond) *Query` | AND 条件を追加 |
| `OrWhereCond(cond Cond) *Query` | OR 条件を追加 |
| `WhereGroup(fn func(*Query)) *Query` | グループ化 AND 条件 |
| `OrWhereGroup(fn func(*Query)) *Query` | グループ化 OR 条件 |

### 順序・ページネーション

| メソッド | 説明 |
|---------|------|
| `OrderBy(order string) *Query` | ORDER BY 句（例: `"CreatedAt DESC"`）|
| `Limit(n int64) *Query` | LIMIT 値 |
| `Offset(n int64) *Query` | OFFSET 値 |
| `ForceIndex(index string) *Query` | `@{FORCE_INDEX=xxx}` ヒント |

### ステートメント生成

| メソッド | 説明 |
|---------|------|
| `ToStatement() (spanner.Statement, error)` | SELECT 文を生成 |
| `ToStmt() (spanner.Statement, error)` | `ToStatement()` のエイリアス |
| `ToCountStatement() (spanner.Statement, error)` | `SELECT COUNT(*)` 文を生成 |

**エラーケース**:
- `"lure_orm: table name is required"` → `From()` 未呼び出し
- `"lure_orm: columns are required"` → `Select()` 未呼び出し

**名前付きパラメータ**: `@p1, @p2, ...` の形式で自動生成。

---

## 5. 条件型（cond.go）

全条件型は `Cond` インターフェースを実装。

### 等値・IN 条件

| 型 | 定義 | SQL 出力 |
|----|------|---------|
| `Eq` | `type Eq map[string]interface{}` | `col = @p1`（複数キー → `AND` でまとめる） |
| `NotEq` | `type NotEq map[string]interface{}` | `col != @p1` |
| `In` | `type In map[string]interface{}` | `col IN UNNEST(@p1)` |
| `NotIn` | `type NotIn map[string]interface{}` | `col NOT IN UNNEST(@p1)` |

### 比較条件

| 型 | SQL 出力 |
|----|---------|
| `Gt` | `col > @p1` |
| `Gte` / `GtOrEq` | `col >= @p1` |
| `Lt` | `col < @p1` |
| `Lte` / `LtOrEq` | `col <= @p1` |

### NULL・パターン条件

| 型 | 定義 | SQL 出力 |
|----|------|---------|
| `IsNull` | `type IsNull []string` | `col IS NULL` |
| `IsNotNull` | `type IsNotNull []string` | `col IS NOT NULL` |
| `Like` | `type Like map[string]string` | `col LIKE @p1` |

### 複合条件

| 型 | 定義 | SQL 出力 |
|----|------|---------|
| `And` | `type And []Cond` | `(cond1 AND cond2 ...)` |
| `Or` | `type Or []Cond` | `(cond1 OR cond2 ...)` |
| `Not` | `type Not struct{ Cond Cond }` | `NOT (condition)` |
| `Raw` | `type Raw struct{ SQL string; Args []interface{} }` | Raw SQL（`?` → `@p1` に自動変換） |

### 重要: 空の `Or` スライス

```go
var where lure_orm.Or
if keyword != "" {
    where = lure_orm.Or{lure_orm.Like{"Name": keyword}}
}
q.Where(where) // 空 Or → WHERE 句なし（全件取得）
```

オプショナルフィルターのパターンとして活用する。

---

## 6. クエリ実行関数（lure_orm.go）

### 型安全な検索（Go Generics）

```go
// 全件取得（なければ空スライス、エラーなし）
func Find[T any](ctx context.Context, txn ReadRunner, q *Query) ([]*T, error)

// 1件取得（なければ nil, nil）
func FindOne[T any](ctx context.Context, txn ReadRunner, q *Query) (*T, error)
```

### 件数・存在確認

```go
func Count(ctx context.Context, txn ReadRunner, q *Query) (int64, error)
func Exists(ctx context.Context, txn ReadRunner, q *Query) (bool, error)
```

### Raw SQL 実行

```go
func QueryAll[T any](ctx context.Context, txn ReadRunner, stmt spanner.Statement) ([]*T, error)
func QueryOne[T any](ctx context.Context, txn ReadRunner, stmt spanner.Statement) (*T, error)
func QueryCount(ctx context.Context, txn ReadRunner, stmt spanner.Statement) (int64, error)
func QueryExists(ctx context.Context, txn ReadRunner, stmt spanner.Statement) (bool, error)
```

### 手動スキャン

```go
// Row 型を返す（ARRAY<STRUCT> 等の複雑型に使用）
func QueryRow(ctx context.Context, txn ReadRunner, stmt spanner.Statement) *Row
```

**Row のメソッド**:

| メソッド | 動作 |
|---------|------|
| `Scan(ptrs ...interface{}) error` | 位置スキャン。なければ `ErrNoRows` |
| `ToStruct(p interface{}) error` | struct マッピング（spanner タグ使用）。なければ `ErrNoRows` |
| `Err() error` | クエリエラーを取得 |

```go
// イテレーター（高度なユースケース）
func ExecuteQuery(ctx context.Context, txn ReadRunner, stmt spanner.Statement) *spanner.RowIterator
// ↑ 呼び出し元が iter.Stop() を呼ぶ必要あり

func IterateRows(ctx context.Context, txn ReadRunner, stmt spanner.Statement, fn func(*spanner.Row) error) error

// PK での単一行読み取り
func ReadRow[T any](ctx context.Context, txn ReadRunner, table string, key spanner.Key, columns []string) (*T, error)
```

### エラー定数

```go
var ErrNoRows = errors.New("lure_orm: no rows in result set")
```

> `FindOne` / `QueryOne` はゼロ件時に `nil, nil` を返す。  
> `QueryRow().Scan()` / `QueryRow().ToStruct()` はゼロ件時に `ErrNoRows` を返す。

---

## 7. ミューテーション関数（lure_orm.go）

### Struct 型ミューテーション（推奨）

#### 単体操作

```go
// INSERT（タイムスタンプ自動設定）
func InsertStruct(ctx context.Context, txn ReadWriteRunner, table string, v interface{}) error

// UPDATE（Original あれば差分 UPDATE、なければ全 UPDATE）
func UpdateStruct(ctx context.Context, txn ReadWriteRunner, table string, v interface{}) error

// UPSERT（Original nil → INSERT、あれば UPDATE）
func InsertOrUpdateStruct(ctx context.Context, txn ReadWriteRunner, table string, v interface{}) error
```

#### バッチ操作（Go Generics）

```go
func InsertStructMulti[T any](ctx context.Context, txn ReadWriteRunner, table string, items []*T) error
func UpdateStructMulti[T any](ctx context.Context, txn ReadWriteRunner, table string, items []*T) error
func InsertOrUpdateStructMulti[T any](ctx context.Context, txn ReadWriteRunner, table string, items []*T) error
```

- 空スライスは `nil` を返す（エラーなし）
- 全 items を 1 回の `BufferWrite` 呼び出しで送信

### プリミティブミューテーション（低レベル）

```go
func Insert(ctx context.Context, txn ReadWriteRunner, table string, columns []string, values []interface{}) error
func Update(ctx context.Context, txn ReadWriteRunner, table string, columns []string, values []interface{}, key spanner.Key) error
func Delete(ctx context.Context, txn ReadWriteRunner, table string, key spanner.Key) error
func DeleteMulti(ctx context.Context, txn ReadWriteRunner, table string, keys []spanner.Key) error
```

### DML 実行

```go
func ExecUpdate(ctx context.Context, txn ReadWriteRunner, stmt spanner.Statement) (int64, error)
// → 影響行数を返す
```

---

## 8. タイムスタンプ管理（timestamps.go）

### 自動管理フィールド

| フィールド名 | 型 | 説明 |
|-----------|-----|------|
| `CreatedAt` | `time.Time` | INSERT 時に `time.Now()` がセットされる |
| `UpdatedAt` | `time.Time` | INSERT / UPDATE 時に `time.Now()` がセットされる |

> `spanner.NullTime` ではなく `time.Time` を使うこと。

### 操作別の挙動

| 操作 | CreatedAt | UpdatedAt | Original 必要 |
|------|-----------|-----------|--------------|
| InsertStruct | `time.Now()` | `time.Now()` | No |
| UpdateStruct | Original から保持 | `time.Now()` | 推奨（なければリセットされる） |
| InsertOrUpdateStruct（Original なし） | `time.Now()` | `time.Now()` | No |
| InsertOrUpdateStruct（Original あり） | Original から保持 | `time.Now()` | Yes |

---

## 9. 差分 UPDATE（Diff-UPDATE）

`UpdateStruct()` が呼ばれたとき、Entity が `EntityWithPK` インターフェースを実装し、かつ `Original` フィールドが非 nil の場合は **差分 UPDATE** を実行する。

### 仕組み

1. `SpannerPrimaryKeyColumns()` で PK カラムを特定
2. `Original` フィールドと現在値を `reflect.DeepEqual` で比較
3. **常に含まれる**: PK カラム + `UpdatedAt`
4. **差分があるカラムのみ含まれる**: 変更されたフィールド
5. 未変更フィールドは UPDATE 対象から除外

### Entity 定義パターン

```go
type User struct {
    UserID    string    `spanner:"UserId" lure_orm:"primary"`
    Email     string    `spanner:"Email"`
    Name      string    `spanner:"Name"`
    Score     int64     `spanner:"Score"`
    CreatedAt time.Time `spanner:"CreatedAt"`
    UpdatedAt time.Time `spanner:"UpdatedAt"`
    Original  *User     `spanner:"-"` // 必須: spanner:"-" タグ
}

func (u *User) SpannerPrimaryKeyColumns() []string {
    return []string{"UserId"}
}
```

### 使用例

```go
// DB から読み込み後に Original をセット
user := &User{UserID: "usr1", Email: "old@test.com", Name: "John", Score: 100, ...}
user.Original = &User{...same values...}

// Email のみ変更
user.Email = "new@test.com"

// 生成される UPDATE:
// UPDATE User SET UserId=@p1, Email=@p2, UpdatedAt=@p3
// ↑ Name と Score は未変更なので含まれない
err := lure_orm.UpdateStruct(ctx, txn, "Users", user)
```

---

## 10. イテレーター（iterator.go）

| 関数 | 説明 |
|------|------|
| `IterateAll[T any](iter *spanner.RowIterator) ([]*T, error)` | 全行を `[]*T` に変換。イテレーターを自動停止 |
| `IterateOne[T any](iter *spanner.RowIterator) (*T, error)` | 最初の行のみ。なければ `nil, nil` |
| `IterateCount(iter *spanner.RowIterator) (int64, error)` | 最初の列から int64 を取得 |
| `IterateExists(iter *spanner.RowIterator) (bool, error)` | 行があれば true |

内部実装: `row.ToStruct(&item)` + `iterator.Done` 処理。

---

## 11. ユーティリティ（utils/utils.go）

### 型ベースユーティリティ

```go
// 型名をテーブル名として返す
func TableName[T any]() string
// → TableName[User]() == "User"

// spanner タグからカラム名一覧をカンマ区切りで返す
func ColumnNames[T any]() string
// → "UserId,Email,Name,Score,CreatedAt,UpdatedAt"

// lure_orm タグ値を取得
func GetFieldTag[T any](fieldName string) string
// → GetFieldTag[User]("UserID") == "primary"

// lure_orm:"primary" タグのカラム名を返す
func FindPrimaryKeyColumn[T any]() string
```

### 型チェック

```go
func IsNullable(v any) bool  // Spanner nullable 型かどうか
func IsTime(v any) bool       // time.Time または spanner.NullTime かどうか
func IsZeroTime(v any) bool   // time 値がゼロ/nil かどうか
```

### 汎用ヘルパー

```go
func Contains[T comparable](slice []T, value T) bool
```

---

## 12. ロガー（logger/）

### LogLevel

```go
const (
    LogLevelNone  = 0  // ログなし
    LogLevelRead  = 1  // 読み取りクエリのみ
    LogLevelWrite = 2  // 書き込みクエリのみ
    LogLevelAll   = 3  // すべて
)
```

### 使用例

```go
log := logger.New(
    logger.WithLogLevel(logger.LogLevelAll),
    logger.WithFields(map[string]any{"service": "my_service"}),
)

log.Read(ctx, "SELECT * FROM Users WHERE Status = @p1")
log.Write(ctx, "INSERT INTO Users (Email) VALUES (@p1)")
log.Error(ctx, err, "database error occurred")
```

---

## 13. struct タグ規則

### `spanner` タグ（カラムマッピング）

```go
type User struct {
    Email string `spanner:"Email"`  // カラム "Email" にマッピング
    Name  string `spanner:"-"`      // Spanner 操作から除外
}
```

### `lure_orm` タグ（ライブラリアノテーション）

| 値 | 説明 |
|----|------|
| `primary` | 主キーフィールドをマーク |
| `create_time` | INSERT 時に時刻を自動セット（非推奨）|
| `update_time` | UPDATE 時に時刻を自動セット（非推奨）|
| `ignore_write` | 書き込み時に無視（現在未使用）|

### Original フィールドの規則

- 名前は `Original` 固定
- 型は自身の struct のポインタ（例: `Original *User`）
- **必ず `spanner:"-"` タグを付けること**（Spanner がスキャンしようとする）

```go
type Article struct {
    ArticleID int64     `spanner:"ArticleId" lure_orm:"primary"`
    Title     string    `spanner:"Title"`
    ViewCount int64     `spanner:"ViewCount"`
    CreatedAt time.Time `spanner:"CreatedAt"`
    UpdatedAt time.Time `spanner:"UpdatedAt"`
    Original  *Article  `spanner:"-"` // ← 必須
}
```

---

## 14. ベストプラクティス

### 1. Struct 型ミューテーションを使う

```go
// ✓ 推奨: 自動タイムスタンプ・差分 UPDATE
lure_orm.InsertStruct(ctx, txn, "Users", &user)
lure_orm.UpdateStruct(ctx, txn, "Users", &user)

// ✗ 非推奨: 手動でカラム/値を管理
lure_orm.Insert(ctx, txn, "Users", cols, vals)
```

### 2. UpdateStruct 前に Original をセット

```go
// DB から読み込んでスナップショットを保存
user := loadFromDB(...)
user.Original = &User{...同じ値...}

// 一部フィールドのみ変更 → 差分 UPDATE
user.Email = "new@test.com"
lure_orm.UpdateStruct(ctx, txn, "Users", &user) // CreatedAt 保持・変更フィールドのみ UPDATE
```

### 3. バッチ操作を使う

```go
// ✓ 推奨: 1 回の BufferWrite
lure_orm.InsertStructMulti(ctx, txn, "Users", items)

// ✗ 非推奨: N 回の BufferWrite
for _, item := range items {
    lure_orm.InsertStruct(ctx, txn, "Users", item)
}
```

### 4. Cond 型で複合条件を組む

```go
// ✓ 推奨: 再利用可能・型安全
q.Where(lure_orm.And{
    lure_orm.Eq{"Status": "active"},
    lure_orm.Or{
        lure_orm.Gt{"Score": 100},
        lure_orm.In{"Level": []int{3, 4, 5}},
    },
})

// ✗ 非推奨: メソッドチェーン（Deprecated）
q.Eq("Status", "active").OrEq("Status", "pending")
```

### 5. オプショナルフィルターに空 Or を使う

```go
var filter lure_orm.Or
if keyword != "" {
    filter = lure_orm.Or{lure_orm.Like{"Name": "%" + keyword + "%"}}
}
q.Where(filter) // 空 Or → WHERE 句なし（全件）
```

### 6. ARRAY<STRUCT> には QueryRow + Scan を使う

```go
// ✗ 不可: QueryOne は内部で ToStruct を使うため ARRAY<STRUCT> 不対応
user, err := lure_orm.QueryOne[User](ctx, txn, stmt)

// ✓ 可能: positional scan
row := lure_orm.QueryRow(ctx, txn, stmt)
err := row.Scan(&item.Field1, &item.Field2, &item.Children) // ARRAY<STRUCT>
```

---

## 15. 既知の制限事項

| 制限 | 説明 |
|------|------|
| `CreatedAt` / `UpdatedAt` は `time.Time` 必須 | `spanner.NullTime` 不可 |
| `Original` は `spanner:"-"` タグ必須 | 付け忘れると Spanner がスキャンしてパニック |
| `QueryOne` / `Find` は ARRAY\<STRUCT\> 非対応 | `QueryRow().Scan()` を使う |
| 差分 UPDATE は `EntityWithPK` + `Original` 両方必須 | 片方欠けると全 UPDATE |
| バッチ操作は all-or-nothing | 1 件失敗でバッチ全体失敗 |
| Lazy loading なし | トランザクション内で即時クエリ実行 |
| 大規模 OR/AND チェーンでパラメーター上限に注意 | Spanner の名前付きパラメーター数制限 |
