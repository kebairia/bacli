package backup

import (
	"fmt"
	"os"
)

func EnsureDirectoryExist(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory %q: %w", dirPath, err)
	}
	return nil
}

func RemoveFile(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file %q: %w", path, err)
	}
	return nil
}
