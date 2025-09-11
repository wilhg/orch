package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wilhg/orch/pkg/store/entstore"
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
	defer res.Body.Close()
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
