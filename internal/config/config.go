package config

import (
	"path/filepath"

	"github.com/takai/htd/internal/model"
)

type Config struct {
	Root string
}

func New(root string) *Config {
	return &Config{Root: root}
}

func (c *Config) DirForKind(kind model.Kind) string {
	return filepath.Join(c.Root, "items", string(kind))
}

func (c *Config) ArchiveItemsDir() string {
	return filepath.Join(c.Root, "archive", "items")
}

func (c *Config) ReferenceDir() string {
	return filepath.Join(c.Root, "reference")
}

func (c *Config) ArchiveReferenceDir() string {
	return filepath.Join(c.Root, "archive", "reference")
}

func (c *Config) AllDirs() []string {
	dirs := make([]string, 0, 9)
	for _, k := range model.ValidKinds() {
		dirs = append(dirs, c.DirForKind(k))
	}
	dirs = append(dirs, c.ArchiveItemsDir(), c.ArchiveReferenceDir(), c.ReferenceDir())
	return dirs
}
