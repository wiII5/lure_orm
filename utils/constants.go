package utils

// Tag keys for struct field annotations.
const (
	// TagSpanner is the tag key for Spanner column name mapping.
	TagSpanner = "spanner"

	// TagLureORM is the tag key for lure_orm specific annotations.
	TagLureORM = "lure_orm"
)

// Tag values for lure_orm annotations.
const (
	// TagPrimary marks a field as the primary key.
	TagPrimary = "primary"

	// TagCreateTime marks a field to be auto-set on insert.
	TagCreateTime = "create_time"

	// TagUpdateTime marks a field to be auto-set on update.
	TagUpdateTime = "update_time"

	// TagDeleteTime marks a field for soft delete tracking.
	TagDeleteTime = "delete_time"

	// TagIgnoreWrite marks a field to be ignored on write operations.
	TagIgnoreWrite = "ignore_write"
)
