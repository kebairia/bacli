package backup

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDirectoryExist(dirPath string) error {
	if err := os.MkdirAll(filepath.Dir(dirPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory %q: %w", dirPath, err)
	}
	return nil
}
