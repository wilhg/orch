package runtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/wilhg/orch/pkg/agent"
	"github.com/wilhg/orch/pkg/store/entstore"
)

// testState is a simple counter-based state used for testing the runtime loop.
type testState struct {
	runID string
	Count int
}

func (s testState) RunID() string      { return s.runID }
func (s testState) Clone() agent.State { return testState{runID: s.runID, Count: s.Count} }

// testReducer applies events and optionally emits intents.
// - "inc" event adds n to Count and emits one "emit_added" intent with n=2
// - "added" event adds n to Count without emitting intents
type testReducer struct{}

func (testReducer) Reduce(ctx context.Context, current agent.State, event agent.Event) (agent.State, []agent.Intent, error) {
	st := current.(testState)
	switch event.Type {
	case "inc":
		// payload: {"n": <int>}
		var p struct {
			N int `json:"n"`
		}
		decode(event.Payload, &p)
		st.Count += p.N
		next := testState{runID: st.runID, Count: st.Count}
		return next, []agent.Intent{{Name: "emit_added", Args: map[string]any{"n": 2}}}, nil
	case "added":
		var p struct {
			N int `json:"n"`
		}
		decode(event.Payload, &p)
		st.Count += p.N
		next := testState{runID: st.runID, Count: st.Count}
		return next, nil, nil
	default:
		return st, nil, nil
	}
}

// testHandler handles the "emit_added" intent by returning one "added" event.
type testHandler struct{}

func (testHandler) CanHandle(intent agent.Intent) bool { return intent.Name == "emit_added" }

func (testHandler) Handle(ctx context.Context, s agent.State, intent agent.Intent) ([]agent.Event, error) {
	n, _ := intent.Args["n"].(int)
	ev := agent.Event{ID: "e-added-" + s.RunID() + "-" + time.Now().UTC().Format("150405.000000"), Type: "added", Timestamp: time.Now().UTC(), Payload: map[string]any{"n": n}}
	return []agent.Event{ev}, nil
}

func decode[T any](payload any, out *T) {
	b, _ := json.Marshal(payload)
	_ = json.Unmarshal(b, out)
}

func TestRunner_ApplyEventThenEffects_SQLite(t *testing.T) {
	ctx := context.Background()

	st, err := entstore.Open(ctx, "sqlite:file:runtime?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	r := NewRunner(st, testReducer{}, []agent.EffectHandler{testHandler{}}, func(runID string) agent.State {
		return testState{runID: runID, Count: 0}
	})

	runID := "run-1"
	// Incoming event: inc n=1
	incoming := agent.Event{ID: "e0", Type: "inc", Timestamp: time.Now().UTC(), Payload: map[string]any{"n": 1}}
	finalState, err := r.HandleEvent(ctx, runID, incoming)
	if err != nil {
		t.Fatalf("HandleEvent error: %v", err)
	}

	// Expect: reducer applied inc(1) and handler emitted added(2). Count = 3
	ts := finalState.(testState)
	if ts.Count != 3 {
		t.Fatalf("final count=%d want 3", ts.Count)
	}

	// Second invocation should rebuild state from stored events and apply again: 3 + (1+2) = 6
	incoming2 := agent.Event{ID: "e3", Type: "inc", Timestamp: time.Now().UTC(), Payload: map[string]any{"n": 1}}
	finalState2, err := r.HandleEvent(ctx, runID, incoming2)
	if err != nil {
		t.Fatalf("HandleEvent #2 error: %v", err)
	}
	ts2 := finalState2.(testState)
	if ts2.Count != 6 {
		t.Fatalf("final count after second run=%d want 6", ts2.Count)
	}
}

func TestRunner_IdempotentIntent_SQLite(t *testing.T) {
	ctx := context.Background()

	st, err := entstore.Open(ctx, "sqlite:file:runtime-idem?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Reducer emits an intent with a fixed idempotency key so duplicates are skipped.
	r := NewRunner(st, reducerWithIdem{}, []agent.EffectHandler{testHandler{}}, func(runID string) agent.State {
		return testState{runID: runID, Count: 0}
	})

	runID := "run-idem"
	ev := agent.Event{ID: "inc1", Type: "inc", Timestamp: time.Now().UTC(), Payload: map[string]any{"n": 1}}
	if _, err := r.HandleEvent(ctx, runID, ev); err != nil {
		t.Fatal(err)
	}
	// Re-send same trigger; reducer will emit the same idempotent intent, which should be skipped.
	if _, err := r.HandleEvent(ctx, runID, ev); err != nil {
		t.Fatal(err)
	}

	// State should be 3 after first (1 + added 2), unchanged after duplicate.
	current, _, err := r.replayState(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	ts := current.(testState)
	if ts.Count != 3 {
		t.Fatalf("count=%d want 3 (idempotent)", ts.Count)
	}
}

type reducerWithIdem struct{}

func (reducerWithIdem) Reduce(ctx context.Context, current agent.State, event agent.Event) (agent.State, []agent.Intent, error) {
	st := current.(testState)
	switch event.Type {
	case "inc":
		var p struct {
			N int `json:"n"`
		}
		decode(event.Payload, &p)
		st.Count += p.N
		next := testState{runID: st.runID, Count: st.Count}
		return next, []agent.Intent{{Name: "emit_added", Args: map[string]any{"n": 2}, IdempotencyKey: "add-two"}}, nil
	case "added":
		var p struct {
			N int `json:"n"`
		}
		decode(event.Payload, &p)
		st.Count += p.N
		return testState{runID: st.runID, Count: st.Count}, nil, nil
	default:
		return st, nil, nil
	}
}
