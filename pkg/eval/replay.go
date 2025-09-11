package eval

import (
	"context"

	"github.com/wilhg/orch/pkg/agent"
	"github.com/wilhg/orch/pkg/runtime"
	"github.com/wilhg/orch/pkg/store"
)

// Capture represents a captured run: initial runID and a sequence of events.
type Capture struct {
	RunID  string        `json:"run_id"`
	Events []agent.Event `json:"events"`
}

// ReplayRun replays a captured run into a fresh runner and returns the final state.
func ReplayRun(ctx context.Context, st store.Store, r agent.Reducer, handlers []agent.EffectHandler, newState runtime.StateFactory, cap Capture, snap runtime.SnapshotCodec) (agent.State, error) {
	rn := runtime.NewRunner(st, r, handlers, newState, runtime.WithSnapshot(snap, 2))
	var final agent.State
	for _, e := range cap.Events {
		s, err := rn.HandleEvent(ctx, cap.RunID, e)
		if err != nil {
			return nil, err
		}
		final = s
	}
	return final, nil
}
