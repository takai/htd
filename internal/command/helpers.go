package command

import (
	"fmt"
	"time"

	"github.com/takai/htd/internal/id"
	"github.com/takai/htd/internal/store"
)

// generateUniqueID returns an ID derived from title at time now, appending a
// numeric suffix (_2, _3, ...) when the base ID already exists anywhere on
// disk (any kind directory or the archive).
func generateUniqueID(c *container, title string, now time.Time) string {
	base := id.Generate(title, now)
	candidate := base
	for i := 2; ; i++ {
		if _, err := store.FindItem(c.cfg, candidate); err != nil && store.IsNotFound(err) {
			break
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
	return candidate
}
