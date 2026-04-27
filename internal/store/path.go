package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
)

// EntityKind discriminates what kind of object a NotFoundError refers to.
type EntityKind string

const (
	EntityItem      EntityKind = "item"
	EntityReference EntityKind = "reference"
)

type NotFoundError struct {
	Kind EntityKind
	ID   string
}

func (e *NotFoundError) Error() string {
	kind := e.Kind
	if kind == "" {
		kind = EntityItem
	}
	return fmt.Sprintf("%s %q not found", kind, e.ID)
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

	return "", &NotFoundError{Kind: EntityItem, ID: id}
}
