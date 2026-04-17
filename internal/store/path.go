package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
)

type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("item %q not found", e.ID)
}

func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

func PathForItem(cfg *config.Config, item *model.Item) string {
	filename := item.ID + ".md"
	if model.IsTerminal(item.Status) {
		return filepath.Join(cfg.ArchiveItemsDir(), filename)
	}
	return filepath.Join(cfg.DirForKind(item.Kind), filename)
}

func FindItem(cfg *config.Config, id string) (string, error) {
	filename := id + ".md"

	// Search active directories
	for _, kind := range model.ValidKinds() {
		p := filepath.Join(cfg.DirForKind(kind), filename)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Search archive
	p := filepath.Join(cfg.ArchiveItemsDir(), filename)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}

	return "", &NotFoundError{ID: id}
}
