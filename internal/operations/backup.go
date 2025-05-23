package operations

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/kebairia/backup/internal/database"
	"github.com/kebairia/backup/internal/logger"
)

func (operator *Operator) BackupDatabase(db database.Database) error {
	start := time.Now()
	backupPath, err := db.Backup()
	complete := time.Now()
	record := NewMetadata(db, start, complete, backupPath, err)
	if err != nil {
		// still write failed metadata
		record.FilePath = "N/A"
		_ = record.Write(filepath.Dir(backupPath))
		return fmt.Errorf("backup failed for %q: %w", db.GetName(), err)
	}

	// Compress the backup file if needed
	if operator.config.Backup.Compress {
		comPath, err := CompressZstd(backupPath)
		if err != nil {
			return fmt.Errorf("compress backup file: %w", err)
		}
		record.FilePath = comPath
	}

	// Write metadata
	record.Write(filepath.Dir(backupPath))
	return nil
}

// BackupAll runs backups for all configured databases in parallel.
func BackupAll(configPath string) error {
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

	var (
		wg   sync.WaitGroup
		errs = make(chan error, len(databases)) // buffered to avoid deadlock
	)

	for _, db := range databases {

		wg.Add(1)
		// start of the goroutine
		go func(db database.Database) {
			// mark this goroutine  as DONE (finished) once this function finish(exit)
			defer wg.Done()

			err := operator.BackupDatabase(db)
			// in case of error, add this error to the error channel
			if err != nil {
				log.Error("backup failed",
					"database", db.GetName(),
					"error", err.Error(),
				)
				errs <- fmt.Errorf("backup failed for %q: %w", db.GetName(), err)
			}
		}(db)
	}

	wg.Wait()
	close(errs)

	// Check for errors
	// for err := range errs {
	// 	return err // return first error
	// }

	return nil
}
