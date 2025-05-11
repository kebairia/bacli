package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
	"github.com/kebairia/backup/internal/vault"
)

func InitializeMongoDBInstance(configPath string) ([]backup.Database, error) {
	var config config.Config
	if err := config.Load(configPath); err != nil {
		return nil, fmt.Errorf("could not parse config %q: %w", configPath, err)
	}
	// Init Vault client
	vaultClient, err := vault.NewClient(config.Vault.Address, config.Vault.Token)
	if err != nil {
		return nil, fmt.Errorf("vault client init: %w", err)
	}
	var dbs []backup.Database
	for _, instance := range config.Mongo.Instances {
		path := fmt.Sprintf("%s/%s", config.Mongo.VaultBasePath, instance.Name)
		role := fmt.Sprintf("mongo-%s-backup", instance.Name) // db.Name == "pg-db1", "mongo-db2", …
		connection, err := vaultClient.FullDBConnection(path, role)
		if err != nil {
			return nil, fmt.Errorf("vault read %q: %w", path, err)
		}
		opts := []backup.MongoDBOption{
			backup.WithMongoCredentials(connection.Username, connection.Password),
			backup.WithMongoDatabase(connection.Database),
			backup.WithMongoHost(connection.Host),
			backup.WithMongoPort(connection.Port),
			backup.WithMongoMethod(instance.Method),
		}
		mg, err := backup.NewMongoDB(config, opts...)
		if err != nil {
			return nil, fmt.Errorf(
				"failed creating MongoDB instance %q: %w",
				connection.Database,
				err,
			)
		}
		dbs = append(dbs, mg)
	}
	return dbs, nil
}

// InitializePostgresInstance loads, parses, and validates the YAML config at configPath.
func InitializePostgresInstance(configPath string) ([]backup.Database, error) {
	var config config.Config
	if err := config.Load(configPath); err != nil {
		return nil, fmt.Errorf("could not parse config %q: %w", configPath, err)
	}
	// Init Vault client
	vaultClient, err := vault.NewClient(config.Vault.Address, config.Vault.Token)
	if err != nil {
		return nil, fmt.Errorf("vault client init: %w", err)
	}
	var dbs []backup.Database
	for _, instance := range config.Postgres.Instances {
		path := fmt.Sprintf("%s/%s", config.Postgres.VaultBasePath, instance.Name)
		role := fmt.Sprintf("pg-%s-backup", instance.Name) // db.Name == "pg-db1", "mongo-db2", …
		connection, err := vaultClient.FullDBConnection(path, role)
		if err != nil {
			return nil, fmt.Errorf("vault read %q: %w", path, err)
		}
		opts := []backup.PostgresOption{
			backup.WithPostgresCredentials(connection.Username, connection.Password),
			backup.WithPostgresDatabase(connection.Database),
			backup.WithPostgresHost(connection.Host),
			backup.WithPostgresPort(connection.Port),
			backup.WithPostgresMethod(instance.Method),
		}
		pg, err := backup.NewPostgres(config, opts...)
		if err != nil {
			return nil, fmt.Errorf(
				"failed creating Postgres instance %q: %w",
				connection.Database,
				err,
			)
		}
		dbs = append(dbs, pg)
	}
	return dbs, nil
}

// InitializeDatabases loads, parses, and validates the YAML config at configPath.
// It constructs and returns a slice of Database instances (Postgres + MongoDB).
func InitializeDatabases(configPath string) ([]backup.Database, error) {
	pg_dbs, err := InitializePostgresInstance(configPath)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres instance: %w", err)
	}
	mg_dbs, err := InitializeMongoDBInstance(configPath)
	if err != nil {
		return nil, fmt.Errorf("initialize mongodb instance: %w", err)
	}

	dbs := make([]backup.Database, 0, len(pg_dbs)+len(mg_dbs))
	dbs = append(dbs, pg_dbs...)
	dbs = append(dbs, mg_dbs...)

	return dbs, nil
}

// BackupDatabase runs a single backup against one Database.
// It returns an error if that backup fails.
func (om *OperationManager) BackupDatabases(
	db backup.Database,
) error {
	start := time.Now()
	record := backup.Metadata{
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

	// Write metadata to file
	record.Write(filepath.Dir(backupPath))

	return nil
}

// RestoreDatabase runs a single restore against one Database.
// It returns an error if that restore fails.
func RestoreDatabase(db backup.Database, record backup.DBRecord) error {
	if err := db.Restore(record.Path); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
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

		// increament my waiting list by one since I'm doing a new backup
		wg.Add(1)
		// start of the goroutine
		go func(db backup.Database) {
			// mark this goroutine  as DONE (finished) once this function finish(exit)
			defer wg.Done()

			// Do the backup
			record, err := BackupDatabase(db)
			// in case of error, add this error to the error channel
			if err != nil {

				log.Error("backup failed",
					"database", db.GetName(),
					"error", err.Error(),
				)
				errs <- fmt.Errorf("backup failed for %q: %w", db.GetName(), err)
			}

			// Since i'm appending to a slice, I need to lock it until I'm finished with it
			mu.Lock()
			// metadata.Backups = append(metadata.Backups, record)
			backupMetas[path].Backups = append(backupMetas[path].Backups, record)
			// unlock it after finish
			mu.Unlock()
		}(db) // call the function right afterward, and give it the current instance as argument
	}

	wg.Wait()
	close(errs)

	// Check for errors
	// for err := range errs {
	// 	return err // return first error
	// }
	for path, meta := range backupMetas {
		err := meta.Write(path)
		if err != nil {
			return fmt.Errorf("failed to write metadata to %q: %w", path, err)
		}

	}

	return nil
}

// RestoreAll reads every metadata.json under Backup.OutputDir/*,
// then, for each successful backup record, finds the matching
// Database instance and restores it, one by one (no goroutines).
func RestoreAll(configPath string) error {
	// 1) Initialize DB instances
	instances, err := InitializeDatabases(configPath)
	if err != nil {
		return fmt.Errorf("initialize databases: %w", err)
	}

	// 2) Reload config to get Backup.OutputDir
	var config config.Config
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}

	if err != nil {
		return fmt.Errorf("glob %q: %w", globPattern, err)
	}
	if len(metaPaths) == 0 {
		return fmt.Errorf("no metadata.json files found under %q", config.Backup.OutputDir)
	}

	record := backup.Metadata{}

			// Match the record’s Name to one of our instances
			var dbInst backup.Database
			for _, inst := range instances {
				if inst.GetName() == rec.Name {
					dbInst = inst
					break
				}
			}
			if dbInst == nil {
				return fmt.Errorf("no database instance named %q", rec.Name)
			}

			// Perform the restore
			if err := RestoreDatabase(dbInst, rec); err != nil {
				return err
			}
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
	}

	// 6) Flush any buffered logs
	logger.Cleanup()
	return nil
}
