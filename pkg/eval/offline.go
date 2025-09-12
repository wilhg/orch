package eval

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

// Fixture represents one prompt evaluation case.
type Fixture struct {
	Name   string         `json:"name"`
	Prompt string         `json:"prompt"`
	Vars   map[string]any `json:"vars"`
	Expect Expectation    `json:"expect"`
}

type Expectation struct {
	Contains    []string `json:"contains,omitempty"`
	NotContains []string `json:"not_contains,omitempty"`
}

// EvaluatePromptFixtures loads fixtures from an fs.FS directory (json files), renders prompts
// with text/template and evaluates basic expectations. Returns score [0,1].
func EvaluatePromptFixtures(fsys fs.FS, dir string) (score float64, total int, passed int, details []string, err error) {
	fixtures, err := loadFixtures(fsys, dir)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	total = len(fixtures)
	if total == 0 {
		return 1, 0, 0, nil, nil
	}
	for _, fx := range fixtures {
		out, rerr := renderTemplate(fx.Prompt, fx.Vars)
		if rerr != nil {
			details = append(details, fx.Name+": render error: "+rerr.Error())
			continue
		}
		ok := true
		for _, s := range fx.Expect.Contains {
			if !strings.Contains(out, s) {
				ok = false
				details = append(details, fx.Name+": missing contains: "+s)
			}
		}
		for _, s := range fx.Expect.NotContains {
			if strings.Contains(out, s) {
				ok = false
				details = append(details, fx.Name+": unexpected contains: "+s)
			}
		}
		if ok {
			passed++
		}
	}
	score = float64(passed) / float64(total)
	return score, total, passed, details, nil
}

func loadFixtures(fsys fs.FS, dir string) ([]Fixture, error) {
	var out []Fixture
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := fs.ReadFile(fsys, filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var fx Fixture
		if err := json.Unmarshal(b, &fx); err != nil {
			return nil, err
		}
		out = append(out, fx)
	}
	return out, nil
}

func renderTemplate(tpl string, vars map[string]any) (string, error) {
	t, err := template.New("p").Option("missingkey=error").Parse(tpl)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}
