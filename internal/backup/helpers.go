package backup

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDirectoryExist(filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}
