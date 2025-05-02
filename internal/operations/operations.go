package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
)

// helper to read KV v2 secrets at path "secret/<engine>/<db>"
func LoadVaultCreds(client *vault.Client, engine, db string) {
}

// InitializeDatabases loads, parses, and validates the YAML config at configPath.
// It constructs and returns a slice of Database instances (Postgres + MongoDB).
func InitializeDatabases(configPath string) ([]backup.Database, error) {
	// Parse configuration file into cfg
	var cfg config.Config
	if err := cfg.LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("could not parse config %q: %w", configPath, err)
	}

	// Retrieve the global logger (must be initialized already)
	log := logger.Global()

	// Prepare slice to hold database instances
	var dbs []backup.Database

	// --- Postgres instances ---
	for _, ps := range cfg.PostgresInstances {
		// Build functional options from config
		opts := []backup.PostgresOption{
			backup.WithPostgresCredentials(ps.Username, ps.Password),
			backup.WithPostgresDatabase(ps.Database),
			backup.WithPostgresHost(ps.Host),
			backup.WithPostgresPort(ps.Port),
			backup.WithPostgresMethod(ps.Method),
		}

		// Create a Postgres backup handler
		pg, err := backup.NewPostgres(cfg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed creating Postgres instance %q: %w", ps.Database, err)
		}

		// Inject the shared logger
		pg.Logger = log

		// Add to the list of databases to back up
		dbs = append(dbs, pg)
	}

	// --- MongoDB instances ---
	for _, m := range cfg.MongoInstances {
		opts := []backup.MongoDBOption{
			backup.WithMongoCredentials(m.Username, m.Password),
			backup.WithMongoDatabase(m.Database),
			backup.WithMongoHost(m.Host),
			backup.WithMongoPort(m.Port),
			backup.WithMongoMethod(m.Method),
		}
		mg, err := backup.NewMongoDB(cfg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed creating MongoDB instance %q: %w", m.Database, err)
		}
		mg.Logger = log
		dbs = append(dbs, mg)
	}

	return dbs, nil
}

// BackupDatabase runs a single backup against one Database.
// It returns an error if that backup fails.
func BackupDatabase(db backup.Database) (backup.DBRecord, error) {
	start := time.Now()
	record := backup.DBRecord{
		Name:      db.GetName(),
		StartedAt: start,
	}
	backupPath, err := db.Backup()
	duration := time.Since(start)
	record.Duration = duration
	if err != nil {
		record.Success = false
		record.Error = err.Error()
		record.Path = "None"
		return record, fmt.Errorf("backup failed for %q: %w", db.GetName(), err)
	}
	record.Success = true
	record.Path = backupPath
	if info, err := os.Stat(backupPath); err == nil {
		record.SizeBytes = info.Size()
	}

	return record, nil
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
	databases, err := InitializeDatabases(configPath)
	if err != nil {
		return fmt.Errorf("initialize databases: %w", err)
	}
	// postgres: [Metadata for postgres dumpfiles]
	backupMetas := map[string]*backup.Metadata{}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs = make(chan error, len(databases)) // buffered to avoid deadlock
	)

	for _, db := range databases {

		path := db.GetPath()
		backupMetas[path] = &backup.Metadata{
			RunAt:   time.Now(),
			Backups: make([]backup.DBRecord, 0, len(databases)),
			Engine:  db.GetEngine(),
		}
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
	var cfg config.Config
	if err := cfg.LoadConfig(configPath); err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}

	// 3) Find all metadata.json files
	globPattern := filepath.Join(cfg.Backup.OutputDir, "*", "metadata.json")
	metaPaths, err := filepath.Glob(globPattern)
	if err != nil {
		return fmt.Errorf("glob %q: %w", globPattern, err)
	}
	if len(metaPaths) == 0 {
		return fmt.Errorf("no metadata.json files found under %q", cfg.Backup.OutputDir)
	}

	var meta backup.Metadata
	// 4) For each metadata.json…
	for _, metaFile := range metaPaths {

		meta.Load(metaFile)
		// 5) …restore each successful record
		for _, rec := range meta.Backups {
			if !rec.Success {
				continue
			}

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
		}
	}

	// 6) Flush any buffered logs
	logger.Cleanup()
	return nil
}
