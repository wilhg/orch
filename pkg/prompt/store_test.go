package prompt

import "testing"

func TestStore_VersioningAndLint(t *testing.T) {
	s := NewStore()

	// lint failure: empty name
	if _, issues, err := s.Save(Prompt{Name: "", Body: "hello"}); err == nil {
		t.Fatal("expected lint failure for missing name")
	} else if len(issues) == 0 {
		t.Fatal("expected issues")
	}

	// save v1
	v1, issues, err := s.Save(Prompt{Name: "welcome", Body: "Hi {{user}}"})
	if err != nil {
		t.Fatalf("save v1: %v (%v)", err, issues)
	}
	if v1.Version != 1 {
		t.Fatalf("v1 version=%d", v1.Version)
	}

	// save v2
	v2, _, err := s.Save(Prompt{Name: "welcome", Body: "Hello {{user}}!"})
	if err != nil {
		t.Fatal(err)
	}
	if v2.Version != 2 {
		t.Fatalf("v2 version=%d", v2.Version)
	}

	got, ok := s.Get("welcome", 0)
	if !ok || got.Version != 2 {
		t.Fatalf("get latest=%+v ok=%v", got, ok)
	}
	got1, ok := s.Get("welcome", 1)
	if !ok || got1.Version != 1 {
		t.Fatalf("get v1=%+v ok=%v", got1, ok)
	}

	all := s.List("welcome")
	if len(all) != 2 || all[0].Version != 1 || all[1].Version != 2 {
		t.Fatalf("list=%+v", all)
	}
}
