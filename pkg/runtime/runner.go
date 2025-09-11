package runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wilhg/orch/pkg/agent"
	"github.com/wilhg/orch/pkg/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StateFactory creates an initial state for a given run.
type StateFactory func(runID string) agent.State

// Runner coordinates reducer execution and effect handling backed by a durable store.
type Runner struct {
	st       store.Store
	reducer  agent.Reducer
	handlers []agent.EffectHandler
	newState StateFactory

	// snapshot settings
	snapshotInterval int
	snapshotCodec    SnapshotCodec
}

// RunnerOption configures the Runner at construction time.
type RunnerOption func(*Runner)

// WithSnapshot enables snapshotting with the provided codec at the given interval (number of events).
// If interval <= 0 or codec is nil, snapshotting is disabled.
func WithSnapshot(codec SnapshotCodec, interval int) RunnerOption {
	return func(r *Runner) {
		if codec != nil && interval > 0 {
			r.snapshotCodec = codec
			r.snapshotInterval = interval
		}
	}
}

// SnapshotCodec encodes/decodes state for durable snapshots.
type SnapshotCodec interface {
	Encode(state agent.State) ([]byte, error)
	Decode(runID string, data []byte) (agent.State, error)
}

// NewRunner constructs a new Runner.
func NewRunner(st store.Store, r agent.Reducer, handlers []agent.EffectHandler, newState StateFactory, opts ...RunnerOption) *Runner {
	rn := &Runner{st: st, reducer: r, handlers: handlers, newState: newState}
	for _, opt := range opts {
		opt(rn)
	}
	return rn
}

// HandleEvent appends the incoming event, replays state, runs the reducer to compute intents,
// dispatches intents to effect handlers, and appends resulting events.
// Returns the final state after processing the entire cycle.
func (r *Runner) HandleEvent(ctx context.Context, runID string, incoming agent.Event) (agent.State, error) {
	tr := otel.Tracer("runtime/runner")
	ctx, span := tr.Start(ctx, "Runner.HandleEvent", trace.WithAttributes(
		attribute.String("run.id", runID),
		attribute.String("event.id", incoming.ID),
		attribute.String("event.type", incoming.Type),
	))
	defer span.End()
	if runID == "" {
		return nil, errors.New("runID is empty")
	}
	if incoming.ID == "" {
		incoming.ID = fmt.Sprintf("e-%s-%d", runID, time.Now().UnixNano())
	}

	// 1) Rebuild state by replaying from latest snapshot + subsequent events.
	current, lastSeq, err := r.replayState(ctx, runID)
	if err != nil {
		return nil, err
	}

	// 2) Apply reducer on the incoming event to get next state and intents.
	// If the incoming event was already recorded (duplicate delivery), skip processing.
	if incoming.ID != "" {
		if _, err := r.st.GetEventByID(ctx, incoming.ID); err == nil {
			return current, nil
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}
	next, intents, err := r.reducer.Reduce(ctx, current, incoming)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	current = next

	// 3) Append the incoming event to durable log after successful reduction.
	if _, err := r.st.AppendEvent(ctx, agentEventToRecord(runID, incoming)); err != nil {
		return nil, err
	}

	// 4) Execute intents via handlers, appending any produced events and applying reducer for each.
	for _, it := range intents {
		handler := r.findHandler(it)
		if handler == nil {
			// skip unknown intents for now; future: log/metric
			continue
		}
		// Idempotency: if intent has IdempotencyKey, skip if a corresponding marker exists.
		if it.IdempotencyKey != "" {
			markerID := intentMarkerEventID(runID, it.IdempotencyKey)
			if _, err := r.st.GetEventByID(ctx, markerID); err == nil {
				// already processed
				continue
			}
		}
		evs, err := handler.Handle(ctx, current, it)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		for _, ev := range evs {
			// append effect event
			if _, err := r.st.AppendEvent(ctx, agentEventToRecord(runID, ev)); err != nil {
				span.RecordError(err)
				return nil, err
			}
			// apply reducer for effect-produced event to update state deterministically
			current, _, err = r.applySingle(ctx, current, ev)
			if err != nil {
				span.RecordError(err)
				return nil, err
			}
			lastSeq++
		}
		// After successful handling, write an idempotency marker event to record completion.
		if it.IdempotencyKey != "" {
			marker := agent.Event{
				ID:        intentMarkerEventID(runID, it.IdempotencyKey),
				Type:      "intent_processed",
				Timestamp: time.Now().UTC(),
				Payload: map[string]any{
					"key":  it.IdempotencyKey,
					"name": it.Name,
				},
			}
			if _, err := r.st.AppendEvent(ctx, agentEventToRecord(runID, marker)); err != nil {
				return nil, err
			}
		}
	}

	// Snapshot policy: snapshot every N events if enabled.
	if r.snapshotCodec != nil && r.snapshotInterval > 0 {
		seq, err := r.st.LastSeq(ctx, runID)
		if err != nil {
			return nil, err
		}
		if seq > 0 && seq%int64(r.snapshotInterval) == 0 {
			if err := r.saveSnapshot(ctx, runID, seq, current); err != nil {
				return nil, err
			}
		}
	}

	return current, nil
}

func (r *Runner) findHandler(it agent.Intent) agent.EffectHandler {
	for _, h := range r.handlers {
		if h.CanHandle(it) {
			return h
		}
	}
	return nil
}

func (r *Runner) replayState(ctx context.Context, runID string) (agent.State, int64, error) {
	// Load snapshot if exists.
	var (
		base agent.State
		upto int64
	)
	base = r.newState(runID)
	sn, err := r.st.LoadLatestSnapshot(ctx, runID)
	if err == nil && len(sn.State) > 0 {
		if r.snapshotCodec != nil {
			if decoded, derr := r.snapshotCodec.Decode(runID, sn.State); derr == nil && decoded != nil {
				base = decoded
			}
		}
		upto = sn.UptoSeq
	}

	events, err := r.st.ListEvents(ctx, runID, upto, 0)
	if err != nil {
		return nil, 0, err
	}
	current := base
	var last int64 = upto
	for _, er := range events {
		ev, err := recordToAgentEvent(er)
		if err != nil {
			return nil, 0, err
		}
		current, _, err = r.applySingle(ctx, current, ev)
		if err != nil {
			return nil, 0, err
		}
		last = er.Seq
	}
	return current, last, nil
}

func (r *Runner) applySingle(ctx context.Context, current agent.State, ev agent.Event) (agent.State, []agent.Intent, error) {
	next, intents, err := r.reducer.Reduce(ctx, current, ev)
	if err != nil {
		return current, nil, err
	}
	return next, intents, nil
}

func agentEventToRecord(runID string, e agent.Event) store.EventRecord {
	var payload json.RawMessage
	if e.Payload != nil {
		b, _ := json.Marshal(e.Payload)
		payload = b
	}
	return store.EventRecord{
		EventID:   e.ID,
		RunID:     runID,
		Type:      e.Type,
		Payload:   payload,
		CreatedAt: e.Timestamp,
	}
}

func recordToAgentEvent(er store.EventRecord) (agent.Event, error) {
	var v any
	if len(er.Payload) > 0 {
		if err := json.Unmarshal(er.Payload, &v); err != nil {
			return agent.Event{}, err
		}
	}
	return agent.Event{
		ID:        er.EventID,
		Type:      er.Type,
		Timestamp: er.CreatedAt,
		Payload:   v,
	}, nil
}

func intentMarkerEventID(runID, key string) string {
	return fmt.Sprintf("intent-%s-%s", runID, key)
}

func (r *Runner) saveSnapshot(ctx context.Context, runID string, upto int64, s agent.State) error {
	if r.snapshotCodec == nil {
		return nil
	}
	data, err := r.snapshotCodec.Encode(s)
	if err != nil {
		return err
	}
	_, err = r.st.SaveSnapshot(ctx, store.SnapshotRecord{
		SnapshotID: fmt.Sprintf("snap-%s-%d", runID, upto),
		RunID:      runID,
		UptoSeq:    upto,
		State:      data,
		CreatedAt:  time.Now().UTC(),
	})
	return err
}
