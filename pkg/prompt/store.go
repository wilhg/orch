package prompt

import (
	"errors"
	"sort"
	"sync"
)

// Prompt represents a versioned prompt artifact.
type Prompt struct {
	Name    string
	Version int
	Body    string
	Meta    map[string]string
}

// Issue describes a lint finding.
type Issue struct {
	Rule    string
	Message string
	Offset  int
}

// Lint runs basic checks on prompts.
func Lint(p Prompt) []Issue {
	var issues []Issue
	if p.Name == "" {
		issues = append(issues, Issue{Rule: "name.required", Message: "name is required"})
	}
	if len(p.Body) == 0 {
		issues = append(issues, Issue{Rule: "body.required", Message: "body is empty"})
	}
	// simple safety check: discourage hardcoded secrets-like patterns
	if containsSecretLike(p.Body) {
		issues = append(issues, Issue{Rule: "security.secrets", Message: "body appears to contain secrets-like content"})
	}
	return issues
}

func containsSecretLike(s string) bool {
	// naive patterns; can be extended
	if len(s) == 0 {
		return false
	}
	if containsAnyFold(s, []string{"aws_secret_access_key", "BEGIN PRIVATE KEY", "sk-"}) {
		return true
	}
	return false
}

func containsAnyFold(s string, needles []string) bool {
	ls := s
	for _, n := range needles {
		if n == "" {
			continue
		}
		if indexFold(ls, n) >= 0 {
			return true
		}
	}
	return false
}

func indexFold(s, sub string) int {
	// simple case-insensitive search
	S := []rune(s)
	U := []rune(sub)
	for i := 0; i+len(U) <= len(S); i++ {
		match := true
		for j := range U {
			a := S[i+j]
			b := U[j]
			if toLower(a) != toLower(b) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func toLower(r rune) rune {
	if 'A' <= r && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// Store is an in-memory versioned prompt store.
type Store struct {
	mu   sync.RWMutex
	data map[string][]Prompt // name -> versions (ascending)
}

func NewStore() *Store { return &Store{data: make(map[string][]Prompt)} }

var ErrLintFailed = errors.New("prompt failed lint checks")

// Save adds a new version. If name exists, version increments by 1; otherwise starts at 1.
// Lint failures return ErrLintFailed with issues via out param.
func (s *Store) Save(p Prompt) (Prompt, []Issue, error) {
	issues := Lint(p)
	if len(issues) > 0 {
		return Prompt{}, issues, ErrLintFailed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := s.data[p.Name]
	next := 1
	if len(versions) > 0 {
		next = versions[len(versions)-1].Version + 1
	}
	np := Prompt{Name: p.Name, Version: next, Body: p.Body, Meta: p.Meta}
	s.data[p.Name] = append(versions, np)
	return np, nil, nil
}

// Get retrieves specific version; if version==0 returns latest.
func (s *Store) Get(name string, version int) (Prompt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions := s.data[name]
	if len(versions) == 0 {
		return Prompt{}, false
	}
	if version <= 0 {
		return versions[len(versions)-1], true
	}
	// versions are ascending; binary search by Version
	i := sort.Search(len(versions), func(i int) bool { return versions[i].Version >= version })
	if i < len(versions) && versions[i].Version == version {
		return versions[i], true
	}
	return Prompt{}, false
}

// List returns all versions for a name in ascending order.
func (s *Store) List(name string) []Prompt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Prompt(nil), s.data[name]...)
	return out
}
