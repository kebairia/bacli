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
func (operator *Operator) RestoreDatabase(db database.Database, record Metadata) error {
	// decompress the file if the compress=true
	if operator.config.Backup.Compression {
		decPath, err := DecompressZstd(record.FilePath)
		if err != nil {
			return err
		}
		// Remove the decompressed files
		defer RemoveFile(decPath)

		record.FilePath = decPath
	}
	if err := db.Restore(record.FilePath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	return nil
}

func RestoreAll(configPath string) error {
	log := logger.Global()
	operator, err := NewOperator(configPath)
	if err != nil {
		return err
	}

	// 1) Initialize DB instances
	databases, err := database.InitializeDatabases(
		operator.ctx,
		operator.config,
		operator.vaultClient,
	)
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
				operator.config.Backup.Directory,
				db.GetEngine(),
				db.GetName(),
				"metadata.json",
			)
			record.Load(metadataFile)

			err := operator.RestoreDatabase(db, record)
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
