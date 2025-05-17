package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kebairia/backup/internal/database"
	"github.com/kebairia/backup/internal/logger"
)

// FIX: write metadata logic is inconsistant, I need to clean it
// I don't think that using `backupPath` is good here

// BackupDatabase runs a single backup against one Database.
// It returns an error if that backup fails.
func (om *OperationManager) BackupDatabase(
	db database.Database,
) error {
	start := time.Now()
	record := Metadata{
		Database:  db.GetName(),
		Engine:    db.GetEngine(),
		StartedAt: start,
	}
	backupPath, err := db.Backup()
	duration := time.Since(start)
	record.CompletedAt = time.Now()
	record.Duration = duration
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FilePath = "None"
		return fmt.Errorf("backup failed for %q: %w", db.GetName(), err)
	}
	record.Status = "success"
	record.FilePath = backupPath
	if info, err := os.Stat(backupPath); err == nil {
		record.SizeBytes = info.Size()
	}

	// Compress the backup file if needed
	if om.cfg.Backup.Compress {
		comPath, err := CompressZstd(backupPath)
		if err != nil {
			return fmt.Errorf("compress backup file: %w", err)
		}
		record.FilePath = comPath
	}

	// Write metadata to file
	record.Write(filepath.Dir(backupPath))

	return nil
}

// BackupAll runs backups for all configured databases in parallel.
func BackupAll(configPath string) error {
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

			err := om.BackupDatabase(db)
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
