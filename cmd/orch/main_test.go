package main

import "testing"

func TestGetEnv(t *testing.T) {
	t.Setenv("FOO", "bar")
	if got := getEnv("FOO", "default"); got != "bar" {
		t.Fatalf("getEnv returned %q, want %q", got, "bar")
	}
	if got := getEnv("MISSING", "default"); got != "default" {
		t.Fatalf("getEnv returned %q, want %q", got, "default")
	}
}
