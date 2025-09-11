package eval

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/wilhg/orch/examples/todo"
	"github.com/wilhg/orch/pkg/agent"
	"github.com/wilhg/orch/pkg/store/entstore"
)

type jsonCodec struct{}

func (jsonCodec) Encode(state agent.State) ([]byte, error) { return json.Marshal(state) }
func (jsonCodec) Decode(runID string, data []byte) (agent.State, error) {
	var s todo.State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Run == "" {
		s.Run = runID
	}
	return s, nil
}

func TestReplayRun_Todo(t *testing.T) {
	ctx := context.Background()
	st, err := entstore.Open(ctx, "sqlite:file:replay?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	cap := Capture{
		RunID: "r1",
		Events: []agent.Event{
			{ID: "e1", Type: "add_task", Timestamp: time.Now().UTC(), Payload: map[string]any{"title": "demo"}},
			{ID: "e2", Type: "complete_task", Timestamp: time.Now().UTC(), Payload: map[string]any{"title": "demo"}},
		},
	}
	final, err := ReplayRun(ctx, st, todo.Reducer{}, []agent.EffectHandler{todo.LoggerEffect{}}, func(runID string) agent.State { return todo.State{Run: runID} }, cap, jsonCodec{})
	if err != nil {
		t.Fatal(err)
	}
	stt := final.(todo.State)
	if stt.Done != 1 {
		t.Fatalf("final Done=%d want 1", stt.Done)
	}
}
