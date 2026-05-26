package core

import "strings"

func BuildPhrase(m ComicMeta) string {
	words := strings.Fields(m.Transcript)
	if len(words) > 50 {
		words = words[:50]
	}
	parts := make([]string, 0, 3)
	if m.Title != "" {
		parts = append(parts, m.Title)
	}
	if m.Alt != "" {
		parts = append(parts, m.Alt)
	}
	if len(words) > 0 {
		parts = append(parts, strings.Join(words, " "))
	}
	return strings.Join(parts, " ")
}
