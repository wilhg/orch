//go:build integration

package entstore

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Test parity by appending the same events to SQLite (in-memory) and Postgres, then
// comparing the final ordered list lengths and sequences. This is a pragmatic check for F05-AC2.
func TestParity_SQLite_vs_Postgres_EventOrdering(t *testing.T) {
	ctx := context.Background()

	// SQLite store
	sqlite, err := Open(ctx, "sqlite:file:parity?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlite.Close() })
	if err := sqlite.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Postgres container
	pgc, err := tcpostgres.RunContainer(ctx,
		tcpostgres.WithDatabase("orch"),
		tcpostgres.WithUsername("orch"),
		tcpostgres.WithPassword("orch"),
		tcpostgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		t.Skipf("skip: cannot start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pg, err := Open(ctx, fmt.Sprintf("postgres://%s", dsn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pg.Close() })
	if err := pg.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]any{"k": "v"})
	// Append the same sequence
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("e%d", i)
		if _, err := sqlite.AppendEvent(ctx, structToEvent(id, "run-parity", "typ", payload)); err != nil {
			t.Fatal(err)
		}
		if _, err := pg.AppendEvent(ctx, structToEvent(id, "run-parity", "typ", payload)); err != nil {
			t.Fatal(err)
		}
	}

	a, err := sqlite.ListEvents(ctx, "run-parity", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	b, err := pg.ListEvents(ctx, "run-parity", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("length mismatch sqlite=%d postgres=%d", len(a), len(b))
	}
	for i := range a {
		if a[i].Seq != b[i].Seq {
			t.Fatalf("seq mismatch at %d: sqlite=%d postgres=%d", i, a[i].Seq, b[i].Seq)
		}
	}
}
