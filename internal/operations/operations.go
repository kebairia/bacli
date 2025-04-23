package operations

import (
	"fmt"

	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/logger"
)

func PerformBackup(db backup.Database) error {
	logger.Init()

	if err := db.Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	return nil
}

func PerformAllBackups(databases []backup.Database) error {
	for _, db := range databases {
		if err := db.Backup(); err != nil {
			return fmt.Errorf("backup failed : %w", err)
		}
	}
	return nil
}
