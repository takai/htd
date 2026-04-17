package store

import (
	"os"

	"github.com/takai/htd/internal/config"
)

func EnsureDirs(cfg *config.Config) error {
	for _, d := range cfg.AllDirs() {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
