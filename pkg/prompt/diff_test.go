package prompt

import "testing"

func TestUnifiedDiff(t *testing.T) {
	a := "Hello\nWorld"
	b := "Hello\nEveryone"
	d := UnifiedDiff(a, b)
	if d == "" || d == a || d == b {
		t.Fatalf("unexpected diff: %q", d)
	}
}

func TestStoreDiff(t *testing.T) {
	s := NewStore()
	p1, _, err := s.Save(Prompt{Name: "x", Body: "A"})
	if err != nil {
		t.Fatal(err)
	}
	p2, _, err := s.Save(Prompt{Name: "x", Body: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if d := s.Diff("x", p1.Version, p2.Version); d == "" {
		t.Fatal("expected diff")
	}
}
