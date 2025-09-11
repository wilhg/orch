package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wilhg/orch/pkg/store"
)

// Option allows configuring DB connection.
type Option func(*config)

type config struct {
	Logger logger.Interface
}

// WithLogger sets a custom GORM logger.
func WithLogger(l logger.Interface) Option { return func(c *config) { c.Logger = l } }

// Open opens a Postgres-backed GORM DB connection using the provided DSN.
func Open(dsn string, opts ...Option) (*Store, error) {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}
	gormCfg := &gorm.Config{}
	if cfg.Logger != nil {
		gormCfg.Logger = cfg.Logger
	}
	db, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&EventModel{}, &SnapshotModel{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// EventModel represents the GORM model for events.
type EventModel struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	RunID     string    `gorm:"index;type:text;not null"`
	Type      string    `gorm:"type:text;not null"`
	Timestamp time.Time `gorm:"index;not null"`
	Payload   []byte    `gorm:"type:bytea;not null"`
}

func (EventModel) TableName() string { return "events" }

// SnapshotModel represents the GORM model for snapshots.
type SnapshotModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	RunID     string    `gorm:"index;uniqueIndex:snap_run_created;type:text;not null"`
	CreatedAt time.Time `gorm:"index:;uniqueIndex:snap_run_created;not null"`
	State     []byte    `gorm:"type:bytea;not null"`
}

func (SnapshotModel) TableName() string { return "snapshots" }

// Store implements store.EventStore and store.SnapshotStore using GORM.
type Store struct{ db *gorm.DB }

// Append inserts a batch of events.
func (s *Store) Append(ctx context.Context, events []store.EventRecord) error {
	if len(events) == 0 {
		return nil
	}
	models := make([]EventModel, 0, len(events))
	for _, e := range events {
		models = append(models, EventModel{
			ID:        e.ID,
			RunID:     e.RunID,
			Type:      e.Type,
			Timestamp: e.Timestamp,
			Payload:   e.Payload,
		})
	}
	return s.db.WithContext(ctx).Create(&models).Error
}

// ListByRun returns all events for a run ordered by timestamp, then id.
func (s *Store) ListByRun(ctx context.Context, runID string) ([]store.EventRecord, error) {
	var models []EventModel
	if err := s.db.WithContext(ctx).Where("run_id = ?", runID).Order("timestamp asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]store.EventRecord, 0, len(models))
	for _, m := range models {
		out = append(out, store.EventRecord{
			ID:        m.ID,
			RunID:     m.RunID,
			Type:      m.Type,
			Timestamp: m.Timestamp,
			Payload:   m.Payload,
		})
	}
	return out, nil
}

// Save stores a snapshot, replacing if newer for the same run.
func (s *Store) Save(ctx context.Context, snap store.SnapshotRecord) error {
	m := SnapshotModel{RunID: snap.RunID, CreatedAt: snap.CreatedAt, State: snap.State}
	return s.db.WithContext(ctx).Create(&m).Error
}

// LoadLatest fetches the most recent snapshot for a run.
func (s *Store) LoadLatest(ctx context.Context, runID string) (store.SnapshotRecord, bool, error) {
	var m SnapshotModel
	err := s.db.WithContext(ctx).Where("run_id = ?", runID).Order("created_at desc").First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return store.SnapshotRecord{}, false, nil
	}
	if err != nil {
		return store.SnapshotRecord{}, false, err
	}
	return store.SnapshotRecord{RunID: m.RunID, CreatedAt: m.CreatedAt, State: m.State}, true, nil
}

// MarshalJSON helpers (optional) to verify payload/state are JSON when needed.
func EnsureJSON(v any) ([]byte, error) { return json.Marshal(v) }
