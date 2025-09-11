package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wilhg/orch/examples/todo"
	"github.com/wilhg/orch/pkg/agent"
	otto "github.com/wilhg/orch/pkg/otel"
	"github.com/wilhg/orch/pkg/runtime"
	"github.com/wilhg/orch/pkg/store"
	"github.com/wilhg/orch/pkg/store/entstore"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	var showVersion bool
	var addr string
	var databaseURL string

	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.StringVar(&addr, "addr", getEnv("ORCH_ADDR", ":8080"), "http listen address")
	flag.StringVar(&databaseURL, "database", getEnv("DATABASE_URL", "sqlite:file:orch.sqlite?_fk=1&cache=shared&_pragma=busy_timeout(5000)"), "database url (sqlite or postgres)")
	flag.Parse()

	if showVersion {
		fmt.Printf("orch %s (commit=%s, date=%s)\n", version, commit, date)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize OpenTelemetry (stdout in dev if ORCH_OTEL_STDOUT=1)
	if shutdown, err := otto.Init(ctx, otto.Config{ServiceName: "orch", UseStdout: os.Getenv("ORCH_OTEL_STDOUT") == "1"}); err == nil {
		defer func() { _ = shutdown(context.Background()) }()
	}

	st, err := entstore.Open(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store open error: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "migrate error: %v\n", err)
		os.Exit(1)
	}

	mux := buildMux(st)

	server := &http.Server{Addr: addr, Handler: otelhttp.NewHandler(mux, "http.server")}
	go func() { _ = server.ListenAndServe() }()
	<-ctx.Done()
	_ = server.Shutdown(context.Background())
}

func buildMux(st store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	// Example: trigger todo reducer/effects through a simple endpoint.
	mux.HandleFunc("/api/examples/todo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			RunID, Type string
			Payload     json.RawMessage
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.RunID == "" || body.Type == "" {
			http.Error(w, "run_id and type required", http.StatusBadRequest)
			return
		}
		runner := runtime.NewRunner(st, todo.Reducer{}, []agent.EffectHandler{todo.LoggerEffect{}}, func(runID string) agent.State { return todo.State{Run: runID} })
		ev := agent.Event{ID: uuid.NewString(), Type: strings.ToLower(body.Type), Timestamp: time.Now().UTC()}
		// decode payload into generic map
		var p any
		_ = json.Unmarshal(body.Payload, &p)
		ev.Payload = p
		s, err := runner.HandleEvent(r.Context(), body.RunID, ev)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, s)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Control plane: create run
	mux.HandleFunc("/api/runs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body struct {
				RunID string `json:"run_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.RunID == "" {
				body.RunID = uuid.NewString()
			}
			// Creating a run is implicit; we persist an initial event for audit.
			rec := store.EventRecord{EventID: uuid.NewString(), RunID: body.RunID, Type: "run_created", CreatedAt: time.Now()}
			if _, err := st.AppendEvent(r.Context(), rec); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"run_id": body.RunID})
		case http.MethodGet:
			// get state: return events and latest snapshot meta
			runID := r.URL.Query().Get("run")
			if runID == "" {
				http.Error(w, "missing run", http.StatusBadRequest)
				return
			}
			events, err := st.ListEvents(r.Context(), runID, 0, 200)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sn, _ := st.LoadLatestSnapshot(r.Context(), runID)
			writeJSON(w, map[string]any{"events": events, "snapshot": sn})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Control plane: pause/resume via event types
	mux.HandleFunc("/api/runs/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			RunID string `json:"run_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RunID == "" {
			http.Error(w, "run_id required", http.StatusBadRequest)
			return
		}
		rec := store.EventRecord{EventID: uuid.NewString(), RunID: body.RunID, Type: "run_paused", CreatedAt: time.Now()}
		if _, err := st.AppendEvent(r.Context(), rec); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/runs/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			RunID string `json:"run_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RunID == "" {
			http.Error(w, "run_id required", http.StatusBadRequest)
			return
		}
		rec := store.EventRecord{EventID: uuid.NewString(), RunID: body.RunID, Type: "run_resumed", CreatedAt: time.Now()}
		if _, err := st.AppendEvent(r.Context(), rec); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			runID := r.URL.Query().Get("run")
			if runID == "" {
				http.Error(w, "missing run", http.StatusBadRequest)
				return
			}
			items, err := st.ListEvents(r.Context(), runID, 0, 100)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, items)
		case http.MethodPost:
			var body struct {
				RunID   string          `json:"run_id"`
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.RunID == "" || body.Type == "" {
				http.Error(w, "run_id and type required", http.StatusBadRequest)
				return
			}
			// Normalize type to lowercase for convention.
			body.Type = strings.ToLower(body.Type)
			rec := store.EventRecord{EventID: uuid.NewString(), RunID: body.RunID, Type: body.Type, Payload: body.Payload, CreatedAt: time.Now()}
			out, err := st.AppendEvent(r.Context(), rec)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, out)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/snapshots", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			runID := r.URL.Query().Get("run")
			if runID == "" {
				http.Error(w, "missing run", http.StatusBadRequest)
				return
			}
			sn, err := st.LoadLatestSnapshot(r.Context(), runID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			writeJSON(w, sn)
		case http.MethodPost:
			var body struct {
				RunID   string          `json:"run_id"`
				UptoSeq int64           `json:"upto_seq"`
				State   json.RawMessage `json:"state"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.RunID == "" {
				http.Error(w, "run_id required", http.StatusBadRequest)
				return
			}
			sn := store.SnapshotRecord{SnapshotID: uuid.NewString(), RunID: body.RunID, UptoSeq: body.UptoSeq, State: body.State, CreatedAt: time.Now()}
			out, err := st.SaveSnapshot(r.Context(), sn)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, out)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func getEnv(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
