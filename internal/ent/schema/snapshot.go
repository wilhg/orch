package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Snapshot holds state materializations.
type Snapshot struct{ ent.Schema }

func (Snapshot) Fields() []ent.Field {
	return []ent.Field{
		field.String("snapshot_id").NotEmpty().Unique(),
		field.String("run_id").NotEmpty(),
		field.Int64("upto_seq").NonNegative(),
		field.JSON("state", map[string]any{}).Optional(),
		field.Time("created_at").Default(time.Now).Immutable().SchemaType(map[string]string{
			dialect.Postgres: "TIMESTAMPTZ",
			dialect.SQLite:   "DATETIME",
		}),
	}
}

func (Snapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id"),
		index.Fields("run_id", "upto_seq").Unique(),
	}
}
