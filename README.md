# lure_orm

Go Generics ベースの Cloud Spanner 向け ORM。lure_server での使用を前提に設計されています。

従来の「ポインタ渡し + エラーのみ返す」スタイルとは異なり、Go Generics を活用して結果を直接返します。

---

## 目次

- [モジュール情報](#モジュール情報)
- [ファイル構成](#ファイル構成)
- [エンティティ構造体の規約](#エンティティ構造体の規約)
- [トランザクションインターフェース](#トランザクションインターフェース)
- [クエリビルダー（SELECT）](#クエリビルダーselect)
- [WHERE 条件](#where-条件)
- [読み取り操作](#読み取り操作)
- [書き込み操作](#書き込み操作)
- [差分ベース UPDATE（Original パターン）](#差分ベース-updateoriginal-パターン)
- [自動タイムスタンプ管理](#自動タイムスタンプ管理)
- [生 SQL クエリ](#生-sql-クエリ)
- [ロギング](#ロギング)
- [テスト](#テスト)

---

## モジュール情報

```
module github.com/wiII5/lure_orm
go 1.23
```

**主要依存:**

| パッケージ | バージョン |
|-----------|-----------|
| `cloud.google.com/go/spanner` | v1.73.0 |
| `cloud.google.com/go` | v0.116.0 |
| `google.golang.org/api` | v0.203.0 |

---

## ファイル構成

```
lure_orm/
├── lure_orm.go       # 公開 API（Find/Insert/Update/Delete 等）
├── cond.go           # WHERE 条件オブジェクト（Cond インターフェース・各型）
├── query.go          # SELECT クエリビルダー（Query 型・メソッドチェーン）
├── transaction.go    # ReadRunner / ReadWriteRunner インターフェース定義
├── timestamps.go     # CreatedAt / UpdatedAt 自動管理・差分検出
├── iterator.go       # クエリ結果 → 構造体変換（Generics）
├── logger/
│   ├── config.go     # LogLevel 設定・Option
│   └── log.go        # Logger 実装（Read / Write / Error）
└── utils/
    ├── constants.go  # spanner / lure_orm タグ定数
    └── utils.go      # 型情報抽出ユーティリティ
```

---

## エンティティ構造体の規約

lure_orm は構造体フィールドのタグを使ってカラム名やフィールドの役割を識別します。

```go
type UserEntity struct {
    UserId    string         `spanner:"UserId"    lure_orm:"primary"`
    Email     string         `spanner:"Email"`
    Status    string         `spanner:"Status"`
    CreatedAt time.Time      `spanner:"CreatedAt"`
    UpdatedAt time.Time      `spanner:"UpdatedAt"`
    Original  *UserEntity    `spanner:"-"`        // DB スナップショット（diff UPDATE 用）
}

// EntityWithPK インターフェースを実装すると差分ベース UPDATE が有効になる
func (e *UserEntity) SpannerPrimaryKeyColumns() []string {
    return []string{"UserId"}
}
```

**タグ一覧:**

| タグ | 値 | 用途 |
|-----|---|-----|
| `spanner` | カラム名 | Spanner カラムとのマッピング |
| `spanner` | `"-"` | マッピング対象外（Original フィールド等） |
| `lure_orm` | `"primary"` | PK フィールドのマーク |
| `lure_orm` | `"create_time"` | INSERT 時に自動設定（CreatedAt） |
| `lure_orm` | `"update_time"` | INSERT/UPDATE 時に自動設定（UpdatedAt） |
| `lure_orm` | `"ignore_write"` | 読み取り専用（書き込み Mutation に含まれない） |

**自動認識されるフィールド名:**

- `CreatedAt time.Time` — InsertStruct 時に `time.Now()` をセット
- `UpdatedAt time.Time` — InsertStruct / UpdateStruct 時に `time.Now()` をセット
- `Original *T` — nil でなければ差分 UPDATE モードが有効になる

---

## トランザクションインターフェース

```go
// 読み取り専用トランザクション用
type ReadRunner interface {
    Query(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator
    Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) *spanner.RowIterator
    ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
}

// 読み書きトランザクション用（ReadRunner を包含）
type ReadWriteRunner interface {
    ReadRunner
    BufferWrite(ms []*spanner.Mutation) error
    Update(ctx context.Context, stmt spanner.Statement) (int64, error)
}
```

- `*spanner.ReadOnlyTransaction` → `ReadRunner` を満たす
- `*spanner.ReadWriteTransaction` → `ReadWriteRunner` を満たす

lure_server では `library/spanner/container.go` が提供するヘルパーでトランザクションを取得し、DAO 層に渡します。

---

## クエリビルダー（SELECT）

`Select(columns).From(table)` を起点にメソッドチェーンでクエリを組み立て、`ToStmt()` で `spanner.Statement` に変換します。

```go
stmt, err := lure_orm.Select("UserId, Email, Status").
    From("Users").
    Where(lure_orm.Eq{"Status": "active"}).
    OrderBy("CreatedAt DESC").
    Limit(20).
    Offset(40).
    ToStmt()
```

**ビルダーメソッド一覧:**

| メソッド | 説明 |
|---------|-----|
| `Select(columns string)` | SELECT 句を指定してクエリを開始 |
| `.From(table string)` | FROM テーブルを指定 |
| `.Column(expr string)` | SELECT 句にカラム式を追加（サブクエリ等） |
| `.Columns(exprs ...string)` | SELECT 句に複数カラム式を追加 |
| `.Where(cond Cond)` | Cond オブジェクトで WHERE 条件を追加（AND） |
| `.OrWhereCond(cond Cond)` | Cond オブジェクトで WHERE 条件を追加（OR） |
| `.WhereRaw(sql string, args ...interface{})` | 生 SQL の WHERE 条件を追加（AND） |
| `.WhereGroup(fn func(*Query))` | グループ括弧付き WHERE 条件を追加（AND） |
| `.OrWhereGroup(fn func(*Query))` | グループ括弧付き WHERE 条件を追加（OR） |
| `.OrderBy(order string)` | ORDER BY 句を指定 |
| `.Limit(n int64)` | LIMIT 句を指定 |
| `.Offset(n int64)` | OFFSET 句を指定 |
| `.ForceIndex(index string)` | `@{FORCE_INDEX=index}` ヒントを付加 |
| `.ToStmt()` | `spanner.Statement` を生成（`ToStatement()` の別名） |
| `.ToCountStatement()` | `SELECT COUNT(*) ...` の `spanner.Statement` を生成 |

**ARRAY サブクエリの追加例:**

```go
q := lure_orm.Select("ProjectId, Title").
    From("Projects").
    Column(fmt.Sprintf(
        `ARRAY(SELECT AS STRUCT %s FROM %s WHERE ProjectId = p.ProjectId) AS Tags`,
        tagColumns, tagTable,
    )).
    Where(lure_orm.Eq{"Status": "active"})
```

**ForceIndex:**

```go
q := lure_orm.Select("*").
    From("Users").
    ForceIndex("UsersByEmail").
    Where(lure_orm.Eq{"Email": email})
// → SELECT * FROM Users@{FORCE_INDEX=UsersByEmail} WHERE Email = @p1
```

---

## WHERE 条件

### Cond インターフェース

全ての条件型は `Cond` インターフェースを実装します。パラメーターは `@p1`, `@p2`, ... の名前付きパラメーターに自動変換されます。

```go
type Cond interface {
    build(paramIndex *int) (sql string, params map[string]interface{})
}
```

### 条件型一覧

| 型 | 生成される SQL |
|---|--------------|
| `Eq{"col": val}` | `col = @p1` |
| `NotEq{"col": val}` | `col != @p1` |
| `In{"col": []T{...}}` | `col IN UNNEST(@p1)` |
| `NotIn{"col": []T{...}}` | `col NOT IN UNNEST(@p1)` |
| `Gt{"col": val}` | `col > @p1` |
| `Gte{"col": val}` （= `GtOrEq`) | `col >= @p1` |
| `Lt{"col": val}` | `col < @p1` |
| `Lte{"col": val}` （= `LtOrEq`) | `col <= @p1` |
| `Like{"col": pattern}` | `col LIKE @p1` |
| `IsNull{"col"}` | `col IS NULL` |
| `IsNotNull{"col"}` | `col IS NOT NULL` |
| `And{cond1, cond2, ...}` | `(cond1 AND cond2 AND ...)` |
| `Or{cond1, cond2, ...}` | `(cond1 OR cond2 OR ...)` |
| `Not{Cond: cond}` | `NOT (cond)` |
| `Raw{SQL: "...", Args: []interface{}{...}}` | SQL をそのまま使用、`?` を名前付きパラメーターに置換 |

**複数キーを持つ場合（Eq/In）:**

```go
// Eq に複数キーを渡すと AND で結合
lure_orm.Eq{"EmailName": "user", "EmailDomain": "example.com"}
// → (EmailName = @p1 AND EmailDomain = @p2)
```

### 使用例

**基本的な AND 条件:**

```go
q := lure_orm.Select("*").From("Users").
    Where(lure_orm.Eq{"Status": "active"}).
    Where(lure_orm.Gte{"Age": 18})
// → WHERE Status = @p1 AND Age >= @p2
```

**OR 条件:**

```go
q := lure_orm.Select("*").From("Users").
    Where(lure_orm.Eq{"Status": "active"}).
    OrWhereCond(lure_orm.Eq{"Status": "pending"})
// → WHERE Status = @p1 OR Status = @p2
```

**And / Or の組み合わせ:**

```go
q := lure_orm.Select("*").From("Users").
    Where(lure_orm.Or{
        lure_orm.And{
            lure_orm.Eq{"Type": "admin"},
            lure_orm.Eq{"Status": "active"},
        },
        lure_orm.And{
            lure_orm.Eq{"Type": "user"},
            lure_orm.In{"Role": []string{"mod", "editor"}},
        },
    })
// → WHERE ((Type = @p1 AND Status = @p2) OR (Type = @p3 AND Role IN UNNEST(@p4)))
```

**NOT:**

```go
q := lure_orm.Select("*").From("Users").
    Where(lure_orm.And{
        lure_orm.Eq{"IsActive": true},
        lure_orm.Not{Cond: lure_orm.In{"Status": []string{"deleted", "suspended"}}},
    })
// → WHERE (IsActive = @p1 AND NOT (Status IN UNNEST(@p2)))
```

**グループ括弧:**

```go
q := lure_orm.Select("*").From("Users").
    Where(lure_orm.Eq{"Type": "user"}).
    WhereGroup(func(sub *lure_orm.Query) {
        sub.Where(lure_orm.Eq{"Status": "active"}).
            OrWhereCond(lure_orm.Eq{"Status": "pending"})
    })
// → WHERE Type = @p1 AND (Status = @p2 OR Status = @p3)
```

**Raw SQL:**

```go
q := lure_orm.Select("*").From("Scores").
    Where(lure_orm.Raw{
        SQL:  "Score BETWEEN ? AND ?",
        Args: []interface{}{10, 100},
    })
// → WHERE Score BETWEEN @p1 AND @p2
```

**N社 × 複数ハッシュの一括検索（Spanner N+1 回避パターン）:**

```go
var orClauses lure_orm.Or
for _, key := range keys {
    orClauses = append(orClauses, lure_orm.And{
        lure_orm.Eq{"CompanyId": key.CompanyId},
        lure_orm.In{"ContentHash": key.ContentHashes},
    })
}
q := lure_orm.Select("*").From("BizReceivedProjects").Where(orClauses)
// → WHERE (CompanyId = @p1 AND ContentHash IN UNNEST(@p2))
//      OR (CompanyId = @p3 AND ContentHash IN UNNEST(@p4)) ...
```

---

## 読み取り操作

### Find / FindOne

```go
// 複数件取得
q := lure_orm.Select("UserId, Email, Status").
    From("Users").
    Where(lure_orm.Eq{"Status": "active"}).
    OrderBy("CreatedAt DESC").
    Limit(20)
users, err := lure_orm.Find[UserEntity](ctx, txn, q)
// → []*UserEntity, error

// 1件取得（見つからなければ nil, nil）
q := lure_orm.Select("*").
    From("Users").
    Where(lure_orm.Eq{"Email": email}).
    Limit(1)
user, err := lure_orm.FindOne[UserEntity](ctx, txn, q)
// → *UserEntity, error
```

### Count / Exists

```go
q := lure_orm.Select("*").From("Users").Where(lure_orm.Eq{"Status": "active"})

// COUNT(*) を実行
count, err := lure_orm.Count(ctx, txn, q)  // int64, error

// 存在チェック（Count > 0 と等価）
exists, err := lure_orm.Exists(ctx, txn, q)  // bool, error
```

### ReadRow（PK による1行取得）

```go
user, err := lure_orm.ReadRow[UserEntity](ctx, txn, "Users",
    spanner.Key{"user-uuid"},
    []string{"UserId", "Email", "Status"})
// → *UserEntity, error
```

---

## 書き込み操作

全ての書き込み操作は Spanner Mutation をバッファリングし、トランザクション Commit 時に一括送信されます。

### Insert

```go
// 列名・値を直接指定（内部で InsertOrUpdate Mutation を使用）
err := lure_orm.Insert(ctx, txn, "Users",
    []string{"UserId", "Email", "Status"},
    []interface{}{userId, email, "active"})

// 構造体から（推奨）。CreatedAt / UpdatedAt を自動設定
user := &UserEntity{UserId: "u1", Email: "test@example.com", Status: "active"}
err := lure_orm.InsertStruct(ctx, txn, "Users", user)

// バッチ挿入
users := []*UserEntity{
    {UserId: "u1", Email: "a@example.com"},
    {UserId: "u2", Email: "b@example.com"},
}
err := lure_orm.InsertStructMulti(ctx, txn, "Users", users)
```

### Update

```go
// 列名・値・PK を直接指定（内部で InsertOrUpdate Mutation を使用）
err := lure_orm.Update(ctx, txn, "Users",
    []string{"UserId", "Email", "UpdatedAt"},
    []interface{}{userId, newEmail, time.Now()},
    spanner.Key{userId})

// 構造体から（全カラム UPDATE）。UpdatedAt を自動設定、CreatedAt は Original から復元
err := lure_orm.UpdateStruct(ctx, txn, "Users", user)

// バッチ更新
err := lure_orm.UpdateStructMulti(ctx, txn, "Users", users)
```

### InsertOrUpdate（UPSERT）

```go
// Original == nil → INSERT（CreatedAt / UpdatedAt を自動設定）
// Original != nil → 差分ベース UPDATE（UpdatedAt を自動設定）
err := lure_orm.InsertOrUpdateStruct(ctx, txn, "Users", user)

// バッチ（INSERT と UPDATE が混在可能）
err := lure_orm.InsertOrUpdateStructMulti(ctx, txn, "Users", items)
```

### Delete

```go
// 単一行
err := lure_orm.Delete(ctx, txn, "Users", spanner.Key{"user-uuid"})

// 複合 PK
err := lure_orm.Delete(ctx, txn, "MessageRooms", spanner.Key{"room-id", "user-id"})

// バッチ削除
keys := []spanner.Key{
    {"u1"}, {"u2"}, {"u3"},
}
err := lure_orm.DeleteMulti(ctx, txn, "Users", keys)
```

### DML（ExecUpdate）

Mutation ではなく DML ステートメントを実行する場合に使います。

```go
stmt := spanner.Statement{
    SQL:    "UPDATE Users SET Status = @status WHERE UserId = @id",
    Params: map[string]interface{}{"status": "inactive", "id": userId},
}
rowsAffected, err := lure_orm.ExecUpdate(ctx, txn, stmt)
```

---

## 差分ベース UPDATE（Original パターン）

`UpdateStruct` / `InsertOrUpdateStruct` は `Original` フィールドが nil でない場合、**変更されたカラムのみ** UPDATE Mutation に含めます。

**条件:**
1. エンティティが `EntityWithPK` インターフェースを実装している（`SpannerPrimaryKeyColumns()` を持つ）
2. `Original` フィールドが nil でない

**常に UPDATE に含まれるもの:**
- PK カラム（必須）
- `UpdatedAt`（自動設定）
- 値が `Original` と異なるカラム

```go
// DB から取得した状態を Original に保持する
original := &UserEntity{UserId: "u1", Email: "old@example.com", Status: "active"}
entity := &UserEntity{
    UserId:   "u1",
    Email:    "new@example.com",  // 変更あり → UPDATE に含まれる
    Status:   "active",           // 変更なし → UPDATE に含まれない
    Original: original,
}

err := lure_orm.UpdateStruct(ctx, txn, "Users", entity)
// → UPDATE Users SET Email = @p1, UpdatedAt = @p2 WHERE UserId = @p3
```

lure_server の `base/` 自動生成 DAO では `SyncOriginal` メソッドを使って Original を一括セットし、バッチ書き込みを効率化しています。

---

## 自動タイムスタンプ管理

| 操作 | CreatedAt | UpdatedAt |
|-----|----------|----------|
| `InsertStruct` / `InsertStructMulti` | `time.Now()` にセット | `time.Now()` にセット |
| `UpdateStruct` / `UpdateStructMulti` | Original.CreatedAt を復元 | `time.Now()` にセット |
| `InsertOrUpdateStruct`（Original なし） | `time.Now()` にセット | `time.Now()` にセット |
| `InsertOrUpdateStruct`（Original あり） | Original.CreatedAt を復元 | `time.Now()` にセット |

- `CreatedAt` / `UpdatedAt` フィールドは `time.Time` 型である必要があります。
- `spanner.NullTime` では自動設定の対象外です。

---

## 生 SQL クエリ

クエリビルダーでは表現しにくい複雑な SQL は `spanner.Statement` を直接渡す関数を使います。

```go
// 複数行取得
stmt := spanner.Statement{
    SQL: `SELECT u.UserId, u.Email,
            (SELECT up.Name FROM UserProfile up WHERE up.UserId = u.UserId) AS Name
          FROM Users u
          WHERE u.Status = @status
          ORDER BY u.CreatedAt DESC
          LIMIT @limit OFFSET @offset`,
    Params: map[string]interface{}{
        "status": "active",
        "limit":  int64(20),
        "offset": int64(0),
    },
}
users, err := lure_orm.QueryAll[UserWithName](ctx, txn, stmt)

// 1行取得（見つからなければ nil, nil）
user, err := lure_orm.QueryOne[UserWithName](ctx, txn, stmt)

// COUNT
count, err := lure_orm.QueryCount(ctx, txn, stmt)

// EXISTS
exists, err := lure_orm.QueryExists(ctx, txn, stmt)
```

### 手動スキャン

```go
// QueryRow：1行を Row ラッパーで取得
row := lure_orm.QueryRow(ctx, txn, stmt)
var userId string
var email string
if err := row.Scan(&userId, &email); err != nil {
    if errors.Is(err, lure_orm.ErrNoRows) {
        // 行なし
    }
    return err
}

// または ToStruct
var entity UserEntity
if err := row.ToStruct(&entity); err != nil { ... }

// ExecuteQuery：RowIterator を直接取得（呼び出し元で Stop() が必要）
iter := lure_orm.ExecuteQuery(ctx, txn, stmt)
defer iter.Stop()

// IterateRows：コールバックで各行を処理
err := lure_orm.IterateRows(ctx, txn, stmt, func(row *spanner.Row) error {
    var entity UserEntity
    return row.ToStruct(&entity)
})

// IterateAll / IterateOne：RowIterator を直接渡す
users, err := lure_orm.IterateAll[UserEntity](iter)
user,  err := lure_orm.IterateOne[UserEntity](iter)
```

**エラー定数:**

```go
var ErrNoRows = errors.New("lure_orm: no rows in result set")
// QueryRow().Scan() / QueryRow().ToStruct() で行が見つからなかった場合に返る
```

---

## ロギング

```go
import "github.com/wiII5/lure_orm/logger"

log := logger.New(
    logger.WithLogLevel(logger.LogLevelAll),
    logger.WithFields(map[string]any{"service": "lure_server", "env": "prod"}),
)

// 読み取りクエリのログ
log.Read(ctx, "SELECT * FROM Users WHERE Status = @p1")

// 書き込みのログ
log.Write(ctx, "INSERT INTO Users ...")

// エラーのログ
log.Error(ctx, err, "failed to fetch user: %s", userId)
```

**LogLevel:**

| 定数 | 値 | 動作 |
|-----|---|-----|
| `LogLevelNone` | 0 | ログ出力なし |
| `LogLevelRead` | 1 | 読み取りのみログ |
| `LogLevelWrite` | 2 | 書き込みのみログ |
| `LogLevelAll` | 3 | 全てログ |

---

## テスト

```bash
go test -v ./tests/...
```

テストは `tests/` ディレクトリ配下にあります。Spanner への実接続は不要で、Mock トランザクション（`mockTxn`）を使ってインメモリで動作します。
