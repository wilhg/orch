package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	otto "github.com/wilhg/orch/pkg/otel"
	"github.com/wilhg/orch/pkg/store/entstore"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("FOO", "bar")
	if got := getEnv("FOO", "default"); got != "bar" {
		t.Fatalf("getEnv returned %q, want %q", got, "bar")
	}
	if got := getEnv("MISSING", "default"); got != "default" {
		t.Fatalf("getEnv returned %q, want %q", got, "default")
	}
}

func TestControlPlane_RunLifecycle(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlite:file:httptest?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	st, err := entstore.Open(t.Context(), "sqlite:file:httptest?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(buildMux(st))
	defer srv.Close()

	// create run
	reqBody := bytes.NewBufferString(`{}`)
	res, err := http.Post(srv.URL+"/api/runs", "application/json", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var created struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.RunID == "" {
		t.Fatal("missing run id")
	}

	// pause
	buf := bytes.NewBufferString(`{"run_id":"` + created.RunID + `"}`)
	res2, err := http.Post(srv.URL+"/api/runs/pause", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("pause status=%d", res2.StatusCode)
	}
	_ = res2.Body.Close()

	// resume
	buf2 := bytes.NewBufferString(`{"run_id":"` + created.RunID + `"}`)
	res3, err := http.Post(srv.URL+"/api/runs/resume", "application/json", buf2)
	if err != nil {
		t.Fatal(err)
	}
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("resume status=%d", res3.StatusCode)
	}
	_ = res3.Body.Close()

	// get state
	res4, err := http.Get(srv.URL + "/api/runs?run=" + created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if res4.StatusCode != http.StatusOK {
		t.Fatalf("get state status=%d", res4.StatusCode)
	}
	_ = res4.Body.Close()
}

func TestHTTPErrorEnvelope_BadJSON(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlite:file:httptest2?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	st, err := entstore.Open(t.Context(), "sqlite:file:httptest2?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(buildMux(st))
	defer srv.Close()

	// Send invalid JSON to /api/events
	res, err := http.Post(srv.URL+"/api/events", "application/json", bytes.NewBufferString("{"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", res.StatusCode)
	}
	var envelope struct {
		Error struct {
			Category string `json:"category"`
			Code     string `json:"code"`
		} `json:"error"`
		TraceID string `json:"trace_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Category != "validation" || envelope.Error.Code != "bad_json" {
		t.Fatalf("unexpected error envelope: %+v", envelope)
	}
}

func TestE2E_TodoExample(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlite:file:e2e?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	st, err := entstore.Open(t.Context(), "sqlite:file:e2e?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(buildMux(st))
	defer srv.Close()

	// Create run
	resRun, err := http.Post(srv.URL+"/api/runs", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var run struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(resRun.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	_ = resRun.Body.Close()
	if run.RunID == "" {
		t.Fatal("missing run id")
	}

	// Drive example: add_task then complete_task
	bodyAdd := `{"RunID":"` + run.RunID + `","Type":"add_task","Payload":{"title":"demo"}}`
	resAdd, err := http.Post(srv.URL+"/api/examples/todo", "application/json", bytes.NewBufferString(bodyAdd))
	if err != nil {
		t.Fatal(err)
	}
	if resAdd.StatusCode != http.StatusOK {
		t.Fatalf("add_task status=%d", resAdd.StatusCode)
	}
	_ = resAdd.Body.Close()

	bodyDone := `{"RunID":"` + run.RunID + `","Type":"complete_task","Payload":{"title":"demo"}}`
	resDone, err := http.Post(srv.URL+"/api/examples/todo", "application/json", bytes.NewBufferString(bodyDone))
	if err != nil {
		t.Fatal(err)
	}
	if resDone.StatusCode != http.StatusOK {
		t.Fatalf("complete_task status=%d", resDone.StatusCode)
	}
	_ = resDone.Body.Close()

	// Events should include run_created, complete_task, logged (order not strictly enforced here)
	resEvents, err := http.Get(srv.URL + "/api/events?run=" + run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resEvents.Body.Close() }()
	var events []struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resEvents.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	var sawLogged, sawComplete bool
	for _, e := range events {
		if e.Type == "complete_task" {
			sawComplete = true
		}
		if e.Type == "logged" {
			sawLogged = true
		}
	}
	if !sawComplete || !sawLogged {
		t.Fatalf("missing expected events: complete=%v logged=%v in %#v", sawComplete, sawLogged, events)
	}
}

func TestErrorEnvelopeIncludesTraceID(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlite:file:trace?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	st, err := entstore.Open(t.Context(), "sqlite:file:trace?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Initialize OTel so that otelhttp creates valid spans with non-empty trace IDs
	shutdown, err := otto.Init(t.Context(), otto.Config{ServiceName: "orch-test", UseStdout: true})
	if err == nil {
		t.Cleanup(func() { _ = shutdown(t.Context()) })
	}
	mux := buildMux(st)
	srv := httptest.NewServer(otelhttp.NewHandler(mux, "http.server"))
	defer srv.Close()

	// bad JSON request to trigger error envelope and ensure trace_id present
	res, err := http.Post(srv.URL+"/api/events", "application/json", bytes.NewBufferString("{"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var envelope struct {
		TraceID string `json:"trace_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.TraceID == "" {
		t.Fatalf("expected trace_id to be set")
	}
}
