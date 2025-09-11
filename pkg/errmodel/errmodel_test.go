package errmodel

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAndFrom(t *testing.T) {
	e := Validation("missing", "field missing", map[string]any{"field": "run_id"})
	if e.Category != CategoryValidation || e.Code != "missing" {
		t.Fatalf("unexpected: %#v", e)
	}
	if got := From(e); got != e {
		t.Fatalf("From should return same error instance")
	}
}

func TestWriteHTTP_StatusAndEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	WriteHTTP(rr, req, Validation("bad_json", "oops", nil))
	if rr.Code != 400 {
		t.Fatalf("status=%d want 400", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "\"category\":\"validation\"") {
		t.Fatalf("body missing category: %s", body)
	}
	if !strings.Contains(body, "\"code\":\"bad_json\"") {
		t.Fatalf("body missing code: %s", body)
	}
}
