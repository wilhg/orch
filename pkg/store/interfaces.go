package store

import (
	"context"
	"encoding/json"
	"time"
)

// EventRecord is the persisted representation of an event.
// Payload holds the event data as JSON.
type EventRecord struct {
	EventID   string
	RunID     string
	Seq       int64
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// SnapshotRecord stores a materialized state up to a given sequence.
type SnapshotRecord struct {
	SnapshotID string
	RunID      string
	UptoSeq    int64
	State      json.RawMessage
	CreatedAt  time.Time
}

// EventStore defines operations for event logs.
type EventStore interface {
	AppendEvent(ctx context.Context, e EventRecord) (EventRecord, error)
	ListEvents(ctx context.Context, runID string, afterSeq int64, limit int) ([]EventRecord, error)
	LastSeq(ctx context.Context, runID string) (int64, error)
}

// SnapshotStore defines operations for reading/writing snapshots.
type SnapshotStore interface {
	SaveSnapshot(ctx context.Context, s SnapshotRecord) (SnapshotRecord, error)
	LoadLatestSnapshot(ctx context.Context, runID string) (SnapshotRecord, error)
}

// Store aggregates event and snapshot stores.
type Store interface {
	EventStore
	SnapshotStore
}
