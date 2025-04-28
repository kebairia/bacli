package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
)

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
func BackupDatabase(db backup.Database) (string, error) {
	backupPath, err := db.Backup()
	if err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}
	return backupPath, nil
}

// RestoreDatabase runs a single restore against one Database.
// It returns an error if that restore fails.
func RestoreDatabase(db backup.Database, dumpFile string) error {
	if err := db.Restore(dumpFile); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	return nil
}

// BackupAll performs backup for all configured databases and writes metadata.
func BackupAll(configPath, metadataPath string) error {
	log, err := logger.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	databases, err := InitializeDatabases(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize databases: %w", err)
	}

	metadata := backup.Metadata{
		RunAt:   time.Now(),
		Backups: make([]backup.DBRecord, 0, len(databases)),
	}

	for _, db := range databases {
		record := backup.DBRecord{
			Name:      db.GetName(),
			StartedAt: time.Now(),
		}

		// path, err := BackupDatabase(db)
		path, err := db.Backup()
		record.Duration = time.Since(record.StartedAt)
		if err != nil {
			record.Success = false
			record.Error = err.Error()
			record.Path = "None"
			log.Error("backup failed",
				"database", db.GetName(),
				"error", err.Error(),
			)
			metadata.Backups = append(metadata.Backups, record)
			continue
		}

		record.Success = true
		record.Path = path
		if info, err := os.Stat(path); err == nil {
			record.SizeBytes = info.Size()
		}
		metadata.Backups = append(metadata.Backups, record)
	}

	if err := metadata.Write(metadataPath); err != nil {
		return fmt.Errorf("failed to write metadata to %q: %w", metadataPath, err)
	}

	return nil
}

// RestoreAll performs restore for all configured databases from the metadata file.
func RestoreAll(configPath, metadataPath string) error {
	log, err := logger.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	databases, err := InitializeDatabases(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize databases: %w", err)
	}

	metadataFile := filepath.Join(metadataPath, "metadata.json")
	var metadata backup.Metadata
	if err := metadata.Load(metadataFile); err != nil {
		return fmt.Errorf("failed to load metadata file %q: %w", metadataFile, err)
	}

	for index, db := range databases {
		record := metadata.Backups[index]
		if !record.Success {
			log.Warn("skipping restore for failed backup",
				"database", record.Name,
				"error", record.Error,
			)
			continue
		}

		if err := db.Restore(record.Path); err != nil {
			return fmt.Errorf("restore failed for %q: %w", record.Name, err)
		}

		log.Info("restore completed",
			"database", record.Name,
			"path", record.Path,
		)
	}

	return nil
}
