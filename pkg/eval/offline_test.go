package eval

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestEvaluatePromptFixtures(t *testing.T) {
	fsys := fstest.MapFS{
		"cases/a.json": {Data: []byte(`{"name":"a","prompt":"Hello {{.name}}","vars":{"name":"Ada"},"expect":{"contains":["Hello Ada"]}}`)},
		"cases/b.json": {Data: []byte(`{"name":"b","prompt":"API key: {{.key}}","vars":{"key":"***"},"expect":{"not_contains":["sk-"]}}`)},
	}
	score, total, passed, details, err := EvaluatePromptFixtures(fsys, "cases")
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || passed != 2 || score != 1 {
		t.Fatalf("score=%v total=%d passed=%d details=%v", score, total, passed, details)
	}

	// missing variable should fail
	fsysFail := fstest.MapFS{
		"cases/x.json": {Data: []byte(`{"name":"x","prompt":"Hello {{.name}}"}`)},
	}
	score2, total2, passed2, _, _ := EvaluatePromptFixtures(fsysFail, "cases")
	if total2 != 1 || passed2 != 0 || score2 != 0 {
		t.Fatalf("expected failure: score=%v total=%d passed=%d", score2, total2, passed2)
	}

	// empty directory -> score 1 with 0 tests
	empty := fstest.MapFS{}
	s3, tot3, pass3, _, err := EvaluatePromptFixtures(empty, "cases")
	if err != nil && !errorsIsNotExist(err) {
		t.Fatalf("unexpected err: %v", err)
	}
	if err == nil && !(s3 == 1 && tot3 == 0 && pass3 == 0) {
		t.Fatalf("empty ok: score=%v total=%d passed=%d", s3, tot3, pass3)
	}
}

func errorsIsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
