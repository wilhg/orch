//go:build integration

package entstore

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/wilhg/orch/pkg/store"
)

func TestPostgresEventFlow(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.RunContainer(ctx,
		tcpostgres.WithDatabase("orch"),
		tcpostgres.WithUsername("orch"),
		tcpostgres.WithPassword("orch"),
		tcpostgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		t.Skipf("skip: cannot start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	var st *Store
	deadline := time.Now().Add(45 * time.Second)
	for {
		st, err = Open(ctx, dsn)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect to postgres after retries: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]any{"k": "v"})
	if _, err := st.AppendEvent(ctx, structToEvent("pe1", "runpg", "typ", payload)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent(ctx, structToEvent("pe2", "runpg", "typ", nil)); err != nil {
		t.Fatal(err)
	}

	// Ensure ordered list.
	got, err := st.ListEvents(ctx, "runpg", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].Seq != 1 || got[1].Seq != 2 {
		t.Fatalf("seq order wrong: %+v", got)
	}

	// Snapshot
	sn, err := st.SaveSnapshot(ctx, store.SnapshotRecord{
		SnapshotID: "snp1",
		RunID:      "runpg",
		UptoSeq:    2,
		State:      payload,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = sn
}
