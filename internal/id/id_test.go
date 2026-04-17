package id_test

import (
	"testing"
	"time"

	"github.com/takai/htd/internal/id"
)

func TestGenerate(t *testing.T) {
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.FixedZone("JST", 9*3600))

	cases := []struct {
		title string
		want  string
	}{
		{"Write the man page", "20260417-write_the_man_page"},
		{"Fix bug #42", "20260417-fix_bug_42"},
		{"Q3 planning", "20260417-q3_planning"},
		{"Hello, World!", "20260417-hello_world"},
		{"  leading and trailing  ", "20260417-leading_and_trailing"},
		{"Multiple   spaces", "20260417-multiple_spaces"},
		{"ALL CAPS TITLE", "20260417-all_caps_title"},
		{"already_has_underscores", "20260417-already_has_underscores"},
		{"dash-separated-words", "20260417-dash_separated_words"},
		{"unicode: café résumé", "20260417-unicode_caf_rsum"},
		{"123 numbers first", "20260417-123_numbers_first"},
		{"!!!punctuation only!!!", "20260417-punctuation_only"},
		{"!!!", "20260417-item"},
		{"", "20260417-item"},
	}

	for _, c := range cases {
		got := id.Generate(c.title, now)
		if got != c.want {
			t.Errorf("Generate(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

func TestGenerateDate(t *testing.T) {
	now := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	got := id.Generate("test", now)
	if got != "20240105-test" {
		t.Errorf("want 20240105-test, got %q", got)
	}
}
