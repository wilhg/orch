package prompt

import (
	"bytes"
	"fmt"
	"strings"
)

// UnifiedDiff returns a simple unified diff between two strings.
func UnifiedDiff(a, b string) string {
	if a == b {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString("--- a\n")
	buf.WriteString("+++ b\n")
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")
	i, j := 0, 0
	for i < len(al) || j < len(bl) {
		if i < len(al) && j < len(bl) && al[i] == bl[j] {
			i++
			j++
			continue
		}
		if i < len(al) {
			fmt.Fprintf(&buf, "-%s\n", al[i])
			i++
		}
		if j < len(bl) {
			fmt.Fprintf(&buf, "+%s\n", bl[j])
			j++
		}
	}
	return buf.String()
}

// Diff returns unified diff between two versions of a prompt name, or empty string if not found.
func (s *Store) Diff(name string, v1, v2 int) string {
	p1, ok1 := s.Get(name, v1)
	p2, ok2 := s.Get(name, v2)
	if !ok1 || !ok2 {
		return ""
	}
	return UnifiedDiff(p1.Body, p2.Body)
}
