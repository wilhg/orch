package entstore

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSQLiteEventAppendAndList(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, "sqlite:file:ent?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]any{"hello": "world"})

	e1, err := st.AppendEvent(ctx, structToEvent("e1", "run1", "type1", payload))
	if err != nil {
		t.Fatal(err)
	}
	if e1.Seq != 1 {
		t.Fatalf("seq=%d want 1", e1.Seq)
	}

	e2, err := st.AppendEvent(ctx, structToEvent("e2", "run1", "type2", nil))
	if err != nil {
		t.Fatal(err)
	}
	if e2.Seq != 2 {
		t.Fatalf("seq=%d want 2", e2.Seq)
	}

	events, err := st.ListEvents(ctx, "run1", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len=%d want 2", len(events))
	}
}
