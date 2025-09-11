// Package store defines persistence interfaces for events and snapshots.
// Implementations must provide identical semantics across backends
// to support deterministic replay and portability.
package store

import (
	"context"
	"time"
)

// EventRecord is the persisted representation of an event.
type EventRecord struct {
	ID        string
	RunID     string
	Type      string
	Timestamp time.Time
	Payload   []byte
}

// SnapshotRecord is a persisted snapshot of state for fast recovery.
type SnapshotRecord struct {
	RunID     string
	CreatedAt time.Time
	State     []byte
}

// EventStore persists and retrieves events for a run.
type EventStore interface {
	Append(ctx context.Context, events []EventRecord) error
	ListByRun(ctx context.Context, runID string) ([]EventRecord, error)
}

// SnapshotStore persists and retrieves snapshots of state.
type SnapshotStore interface {
	Save(ctx context.Context, snap SnapshotRecord) error
	LoadLatest(ctx context.Context, runID string) (SnapshotRecord, bool, error)
}
