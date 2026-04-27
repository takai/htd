package command

import (
	"fmt"
	"time"

	"github.com/takai/htd/internal/id"
	"github.com/takai/htd/internal/store"
)

// generateUniqueID returns an ID derived from title at time now, appending a
// numeric suffix (_2, _3, ...) when the base ID already exists anywhere on
// disk: any item directory or archive, or any reference tool directory or
// archive. Cross-checking references keeps IDs globally unique per
// docs/datamodel.md §4.2.
func generateUniqueID(c *container, title string, now time.Time) string {
	base := id.Generate(title, now)
	candidate := base
	for i := 2; ; i++ {
		if !idInUse(c, candidate) {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
}

func idInUse(c *container, candidate string) bool {
	if _, err := store.FindItem(c.cfg, candidate); err == nil {
		return true
	}
	if store.ReferenceExists(c.cfg, candidate) {
		return true
	}
	return false
}
