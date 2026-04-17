package id

import (
	"strings"
	"time"
	"unicode"
)

func Generate(title string, now time.Time) string {
	date := now.Format("20060102")
	slug := toSlug(title)
	if slug == "" {
		slug = "item"
	}
	return date + "-" + slug
}

func toSlug(title string) string {
	s := strings.ToLower(title)
	var b strings.Builder
	prevUnderscore := true // treat start as if preceded by underscore to avoid leading underscore
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			prevUnderscore = false
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			if !prevUnderscore {
				b.WriteRune('_')
				prevUnderscore = true
			}
		}
		// non-alphanumeric, non-separator characters are dropped
	}
	result := b.String()
	// trim trailing underscore
	result = strings.TrimRight(result, "_")
	return result
}
