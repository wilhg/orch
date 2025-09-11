package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Event holds the schema definition for the Event entity.
type Event struct{ ent.Schema }

// Fields of the Event.
func (Event) Fields() []ent.Field {
	return []ent.Field{
		// External stable ID for idempotency and traceability.
		field.String("event_id").NotEmpty().Unique(),
		field.String("run_id").NotEmpty(),
		// Monotonic sequence per run.
		field.Int64("seq").NonNegative(),
		field.String("type").NotEmpty(),
		// JSON payload; compatible with Postgres (JSONB) and SQLite (TEXT/BLOB).
		field.JSON("payload", map[string]any{}).
			Optional(),
		field.Time("created_at").Default(time.Now).Immutable().SchemaType(map[string]string{
			dialect.Postgres: "TIMESTAMPTZ",
			dialect.SQLite:   "DATETIME",
		}),
	}
}

// Indexes of the Event.
func (Event) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id", "seq").Unique(),
		index.Fields("run_id"),
	}
}
