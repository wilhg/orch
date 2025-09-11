package todo

import (
	"context"
	"time"

	"github.com/wilhg/orch/pkg/agent"
)

// State holds a simple counter representing todo items done.
type State struct {
	Run  string `json:"run"`
	Done int    `json:"done"`
}

func (s State) RunID() string      { return s.Run }
func (s State) Clone() agent.State { return State{Run: s.Run, Done: s.Done} }

type Reducer struct{}

// Events:
// - add_task {title}
// - complete_task {title}
// For demo, completing a task emits an intent to log.
func (Reducer) Reduce(ctx context.Context, current agent.State, ev agent.Event) (agent.State, []agent.Intent, error) {
	st := current.(State)
	switch ev.Type {
	case "add_task":
		return st, nil, nil
	case "complete_task":
		st.Done++
		return st, []agent.Intent{{Name: "log", Args: map[string]any{"msg": "task completed"}, IdempotencyKey: ev.ID + "-log"}}, nil
	default:
		return st, nil, nil
	}
}

type LoggerEffect struct{}

func (LoggerEffect) CanHandle(it agent.Intent) bool { return it.Name == "log" }

func (LoggerEffect) Handle(ctx context.Context, s agent.State, it agent.Intent) ([]agent.Event, error) {
	// Emit a synthetic event acknowledging log.
	return []agent.Event{{ID: "ack-" + time.Now().UTC().Format("150405.000000"), Type: "logged", Timestamp: time.Now().UTC(), Payload: it.Args}}, nil
}
