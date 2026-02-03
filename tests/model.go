package tests

import (
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
)

// User is a test model representing a user account.
type User struct {
	UserID     int64             `spanner:"UserId" lure_orm:"primary"`
	Email      string            `spanner:"Email"`
	Name       spanner.NullString `spanner:"Name"`
	Status     string            `spanner:"Status"`
	Age        spanner.NullInt64 `spanner:"Age"`
	Score      float64           `spanner:"Score"`
	IsActive   bool              `spanner:"IsActive"`
	Tags       []string          `spanner:"Tags"`
	CreateTime time.Time         `spanner:"CreateTime" lure_orm:"create_time"`
	UpdateTime time.Time         `spanner:"UpdateTime" lure_orm:"update_time"`
	DeleteTime spanner.NullTime  `spanner:"DeleteTime" lure_orm:"delete_time"`
}

// Article is a test model representing an article.
type Article struct {
	ArticleID   int64            `spanner:"ArticleId" lure_orm:"primary"`
	AuthorID    int64            `spanner:"AuthorId"`
	Title       string           `spanner:"Title"`
	Content     string           `spanner:"Content"`
	PublishedAt spanner.NullTime `spanner:"PublishedAt"`
	ViewCount   int64            `spanner:"ViewCount" lure_orm:"ignore_write"`
	CreateTime  time.Time        `spanner:"CreateTime" lure_orm:"create_time"`
	UpdateTime  time.Time        `spanner:"UpdateTime" lure_orm:"update_time"`
	DeleteTime  spanner.NullTime `spanner:"DeleteTime" lure_orm:"delete_time"`
}

// DataTypeTest is a model for testing various Spanner data types.
type DataTypeTest struct {
	ID           int64               `spanner:"Id" lure_orm:"primary"`
	StringVal    string              `spanner:"StringVal"`
	NullString   spanner.NullString  `spanner:"NullString"`
	Int64Val     int64               `spanner:"Int64Val"`
	NullInt64    spanner.NullInt64   `spanner:"NullInt64"`
	Float64Val   float64             `spanner:"Float64Val"`
	NullFloat64  spanner.NullFloat64 `spanner:"NullFloat64"`
	BoolVal      bool                `spanner:"BoolVal"`
	NullBool     spanner.NullBool    `spanner:"NullBool"`
	DateVal      civil.Date          `spanner:"DateVal"`
	NullDate     spanner.NullDate    `spanner:"NullDate"`
	TimestampVal time.Time           `spanner:"TimestampVal"`
	NullTime     spanner.NullTime    `spanner:"NullTime"`
	StringArray  []string            `spanner:"StringArray"`
	Int64Array   []int64             `spanner:"Int64Array"`
	Float64Array []float64           `spanner:"Float64Array"`
}
