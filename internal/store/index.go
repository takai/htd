package store

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/takai/htd/internal/config"
)

// IndexHeader is the H1 line at the top of every generated INDEX.md.
const IndexHeader = "# Reference index"

// IndexEmptyMarker is the body line written when no references exist.
const IndexEmptyMarker = "_No entries._"

// indexCanonicalTypes is the fixed render order for type:* tag sections.
// Anything not in this list (including unrecognized type:* values and
// references with no type:* tag at all) falls into the trailing "other"
// section.
var indexCanonicalTypes = []string{"user", "feedback", "area_of_focus", "project", "reference"}

// indexOtherSection is the header for entries that do not belong to any of
// the canonical type sections.
const indexOtherSection = "other"

// indexDescRuneLimit caps the description portion of each INDEX.md line.
// A short cap keeps lines diff-friendly and predictable; bodies routinely
// exceed this for the "How to apply" section, so the limit is real.
const indexDescRuneLimit = 80

// RenderIndex returns the INDEX.md byte stream for refs. The output is fully
// deterministic: the same input — both content and slice order — produces the
// same bytes.
//
// Sections are emitted in canonical order: user, feedback, project,
// reference, other. Within each section entries sort by UpdatedAt desc with
// ID asc as the tiebreaker. Empty sections are omitted; if every section is
// empty, an empty-state marker is written.
func RenderIndex(refs []ReferenceWithBody) []byte {
	groups := groupByType(refs)

	var sb strings.Builder
	sb.WriteString(IndexHeader)
	sb.WriteString("\n")

	wroteAny := false
	for _, name := range append(append([]string{}, indexCanonicalTypes...), indexOtherSection) {
		entries := groups[name]
		if len(entries) == 0 {
			continue
		}
		sortReferences(entries)
		sb.WriteString("\n## ")
		sb.WriteString(name)
		sb.WriteString("\n\n")
		for _, e := range entries {
			sb.WriteString(formatIndexLine(e))
			sb.WriteString("\n")
		}
		wroteAny = true
	}
	if !wroteAny {
		sb.WriteString("\n")
		sb.WriteString(IndexEmptyMarker)
		sb.WriteString("\n")
	}
	return []byte(sb.String())
}

// WriteIndex regenerates reference/<tool>/INDEX.md from the current set of
// active references owned by tool.
func WriteIndex(cfg *config.Config, tool string) error {
	refs, err := ListReferences(cfg, tool, false)
	if err != nil {
		return err
	}
	if err := EnsureReferenceToolDir(cfg, tool); err != nil {
		return err
	}
	path := filepath.Join(cfg.ReferenceToolDir(tool), "INDEX.md")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, RenderIndex(refs), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// groupByType buckets refs by their type:* tag (canonical only) or "other".
// A reference with multiple recognized type:* tags is placed in the first
// canonical match; this is documented behavior and keeps each entry in
// exactly one section.
func groupByType(refs []ReferenceWithBody) map[string][]ReferenceWithBody {
	groups := map[string][]ReferenceWithBody{}
	for _, r := range refs {
		section := indexOtherSection
		for _, t := range indexCanonicalTypes {
			if hasRefTag(r.Reference.Tags, "type:"+t) {
				section = t
				break
			}
		}
		groups[section] = append(groups[section], r)
	}
	return groups
}

func hasRefTag(tags []string, want string) bool {
	return slices.Contains(tags, want)
}

// formatIndexLine returns one bullet line. Format:
//
//   - [title](id.md) — short description
//
// The em-dash and description are omitted when no description is available.
func formatIndexLine(r ReferenceWithBody) string {
	title := r.Reference.Title
	id := r.Reference.ID
	desc := extractDescription(r.Body)
	if desc == "" {
		return "- [" + title + "](" + id + ".md)"
	}
	return "- [" + title + "](" + id + ".md) — " + desc
}

// extractDescription returns the first non-blank line of body, with leading
// ATX heading hashes stripped, trimmed to a fixed rune limit. Returns the
// empty string when no usable line exists.
func extractDescription(body string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Strip leading "#" runs from ATX headings.
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return truncateRunesIndex(line, indexDescRuneLimit)
	}
	return ""
}

func truncateRunesIndex(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
