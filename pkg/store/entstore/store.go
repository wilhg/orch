// Package entstore provides an ent-backed implementation of the store interfaces
// compatible with both PostgreSQL and SQLite.
package entstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/wilhg/orch/internal/ent"
	"github.com/wilhg/orch/internal/ent/event"
	"github.com/wilhg/orch/internal/ent/snapshot"
	"github.com/wilhg/orch/pkg/store"
)

// Store implements store.Store backed by ent and supports PostgreSQL and SQLite.
type Store struct {
	client *ent.Client
}

// Open opens an ent client using a DATABASE_URL style DSN.
// Examples:
//   - postgres:  postgres://user:pass@host:5432/dbname?sslmode=disable
//   - sqlite:    sqlite:file:./db.sqlite?cache=shared&_pragma=busy_timeout(5000)
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("databaseURL is empty")
	}
	var (
		drvName string
		dsn     string
		dialect string
	)
	lower := strings.ToLower(databaseURL)
	if strings.HasPrefix(lower, "sqlite:") {
		// ncruces/go-sqlite3 uses driver name "sqlite3" and DSN like file:... or :memory:
		drvName = "sqlite3"
		dsn = strings.TrimPrefix(databaseURL, "sqlite:")
		if dsn == "" {
			dsn = "file:orch.sqlite?cache=shared&_pragma=busy_timeout(5000)"
		}
		// ent expects sqlite3 dialect token for sqlite family
		dialect = "sqlite3"
	} else {
		// Support both URL-style and keyword-style DSNs for pgx.
		u, err := url.Parse(databaseURL)
		if err == nil && u.Scheme != "" {
			switch strings.ToLower(u.Scheme) {
			case "postgres", "postgresql":
				drvName = "pgx"
				dsn = databaseURL
				dialect = "postgres"
			default:
				return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
			}
		} else {
			// Keyword-style DSN (e.g., "user=... password=... host=... dbname=...")
			if strings.Contains(databaseURL, "host=") || strings.Contains(databaseURL, "user=") || strings.Contains(databaseURL, "dbname=") {
				drvName = "pgx"
				dsn = databaseURL
				dialect = "postgres"
			} else {
				return nil, fmt.Errorf("unsupported dsn format")
			}
		}
	}

	db, err := sql.Open(drvName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	drv := entsql.OpenDB(dialect, db)
	client := ent.NewClient(ent.Driver(drv))
	return &Store{client: client}, nil
}

// Migrate creates or updates the database schema.
func (s *Store) Migrate(ctx context.Context) error {
	return s.client.Schema.Create(ctx)
}

// Close closes the underlying client.
func (s *Store) Close() error { return s.client.Close() }

// AppendEvent appends a new event with an incremented sequence per run.
func (s *Store) AppendEvent(ctx context.Context, e store.EventRecord) (store.EventRecord, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return store.EventRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// Find current max seq for this run.
	var nextSeq int64 = 1
	last, err := tx.Event.
		Query().
		Where(event.RunID(e.RunID)).
		Order(ent.Desc(event.FieldSeq)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return store.EventRecord{}, err
	}
	if err == nil && last != nil {
		nextSeq = last.Seq + 1
	}

	var payload map[string]any
	if len(e.Payload) > 0 {
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return store.EventRecord{}, fmt.Errorf("invalid payload json: %w", err)
		}
	}

	b := tx.Event.
		Create().
		SetEventID(e.EventID).
		SetRunID(e.RunID).
		SetSeq(nextSeq).
		SetType(e.Type).
		SetCreatedAt(time.Now())
	if payload != nil {
		b = b.SetPayload(payload)
	}
	created, err := b.Save(ctx)
	if err != nil {
		// If duplicate event_id, return existing record (idempotent append).
		if ent.IsConstraintError(err) {
			existing, gerr := tx.Event.Query().Where(event.EventID(e.EventID)).First(ctx)
			if gerr == nil {
				var raw2 json.RawMessage
				if existing.Payload != nil {
					bb, _ := json.Marshal(existing.Payload)
					raw2 = bb
				}
				return store.EventRecord{
					EventID:   existing.EventID,
					RunID:     existing.RunID,
					Seq:       existing.Seq,
					Type:      existing.Type,
					Payload:   raw2,
					CreatedAt: existing.CreatedAt,
				}, nil
			}
		}
		return store.EventRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.EventRecord{}, err
	}

	var raw json.RawMessage
	if created.Payload != nil {
		b, _ := json.Marshal(created.Payload)
		raw = b
	}
	return store.EventRecord{
		EventID:   created.EventID,
		RunID:     created.RunID,
		Seq:       created.Seq,
		Type:      created.Type,
		Payload:   raw,
		CreatedAt: created.CreatedAt,
	}, nil
}

// ListEvents lists events for a run after a given sequence.
func (s *Store) ListEvents(ctx context.Context, runID string, afterSeq int64, limit int) ([]store.EventRecord, error) {
	q := s.client.Event.Query().Where(event.RunID(runID))
	if afterSeq > 0 {
		q = q.Where(event.SeqGT(afterSeq))
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	rows, err := q.Order(ent.Asc(event.FieldSeq)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.EventRecord, 0, len(rows))
	for _, r := range rows {
		var raw json.RawMessage
		if r.Payload != nil {
			b, _ := json.Marshal(r.Payload)
			raw = b
		}
		out = append(out, store.EventRecord{
			EventID:   r.EventID,
			RunID:     r.RunID,
			Seq:       r.Seq,
			Type:      r.Type,
			Payload:   raw,
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

// LastSeq returns the last sequence for a run.
func (s *Store) LastSeq(ctx context.Context, runID string) (int64, error) {
	rec, err := s.client.Event.Query().
		Where(event.RunID(runID)).
		Order(ent.Desc(event.FieldSeq)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	return rec.Seq, nil
}

// GetEventByID looks up an event by its stable EventID.
func (s *Store) GetEventByID(ctx context.Context, eventID string) (store.EventRecord, error) {
	rec, err := s.client.Event.Query().
		Where(event.EventID(eventID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.EventRecord{}, sql.ErrNoRows
		}
		return store.EventRecord{}, err
	}
	var raw json.RawMessage
	if rec.Payload != nil {
		b, _ := json.Marshal(rec.Payload)
		raw = b
	}
	return store.EventRecord{
		EventID:   rec.EventID,
		RunID:     rec.RunID,
		Seq:       rec.Seq,
		Type:      rec.Type,
		Payload:   raw,
		CreatedAt: rec.CreatedAt,
	}, nil
}

// SaveSnapshot saves a snapshot; unique per (run_id, upto_seq).
func (s *Store) SaveSnapshot(ctx context.Context, sn store.SnapshotRecord) (store.SnapshotRecord, error) {
	var state map[string]any
	if len(sn.State) > 0 {
		if err := json.Unmarshal(sn.State, &state); err != nil {
			return store.SnapshotRecord{}, fmt.Errorf("invalid state json: %w", err)
		}
	}
	sb := s.client.Snapshot.Create().
		SetSnapshotID(sn.SnapshotID).
		SetRunID(sn.RunID).
		SetUptoSeq(sn.UptoSeq).
		SetCreatedAt(time.Now())
	if state != nil {
		sb = sb.SetState(state)
	}
	created, err := sb.Save(ctx)
	if err != nil {
		return store.SnapshotRecord{}, err
	}
	var raw json.RawMessage
	if created.State != nil {
		b, _ := json.Marshal(created.State)
		raw = b
	}
	return store.SnapshotRecord{
		SnapshotID: created.SnapshotID,
		RunID:      created.RunID,
		UptoSeq:    created.UptoSeq,
		State:      raw,
		CreatedAt:  created.CreatedAt,
	}, nil
}

// LoadLatestSnapshot loads the latest snapshot for the run.
func (s *Store) LoadLatestSnapshot(ctx context.Context, runID string) (store.SnapshotRecord, error) {
	rec, err := s.client.Snapshot.Query().
		Where(snapshot.RunID(runID)).
		Order(ent.Desc(snapshot.FieldUptoSeq)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.SnapshotRecord{}, sql.ErrNoRows
		}
		return store.SnapshotRecord{}, err
	}
	var raw json.RawMessage
	if rec.State != nil {
		b, _ := json.Marshal(rec.State)
		raw = b
	}
	return store.SnapshotRecord{
		SnapshotID: rec.SnapshotID,
		RunID:      rec.RunID,
		UptoSeq:    rec.UptoSeq,
		State:      raw,
		CreatedAt:  rec.CreatedAt,
	}, nil
}
