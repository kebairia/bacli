package operations

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/kebairia/backup/internal/database"
	"github.com/kebairia/backup/internal/logger"
)

// RestoreDatabase runs a single restore against one Database.
// It returns an error if that restore fails.
// NOTE: Check for metadata.json in the backup directory,
// NOTE: if not exist, return an error
func (om *OperationManager) RestoreDatabase(db database.Database, record Metadata) error {
	// decompress the file if the compress=true
	if om.cfg.Backup.Compress {
		decPath, err := DecompressZstd(record.FilePath)
		if err != nil {
			return err
		}
		record.FilePath = decPath

		// Remove the decompressed files
		defer RemoveFile(record.FilePath)
	}
	if err := db.Restore(record.FilePath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	return nil
}

func RestoreAll(configPath string) error {
	log := logger.Global()
	om, err := NewOperationManager(configPath)
	if err != nil {
		return err
	}

	// 1) Initialize DB instances
	databases, err := om.InitializeDatabases()
	if err != nil {
		return fmt.Errorf("initialize databases: %w", err)
	}

	record := Metadata{}
	var wg sync.WaitGroup

	for _, db := range databases {
		wg.Add(1)

		go func(db database.Database, record Metadata) {
			defer wg.Done()
			// increament my waiting list by one since I'm doing a new backup
			metadataFile := filepath.Join(
				om.cfg.Backup.OutputDirectory,
				db.GetEngine(),
				db.GetName(),
				"metadata.json",
			)
			record.Load(metadataFile)

			err := om.RestoreDatabase(db, record)
			// in case of error, add this error to the error channel
			if err != nil {
				log.Error("restore failed",
					"database", db.GetName(),
					"error", err.Error(),
				)
			}
		}(db, record)
	}
	wg.Wait()
	return nil
}
