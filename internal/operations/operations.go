package operations

import (
	"fmt"

	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
)

// InitializeDatabases reads the YAML config at configPath, sets up
// a Zap logger, and returns a slice of backup.Database instances
// (Postgres + MongoDB) according to that config.
func InitializeDatabases(configPath string) ([]backup.Database, error) {
	// 1. Initialize structured logger (zap)
	log, err := logger.Init()
	if err != nil {
		return nil, fmt.Errorf("logger init failed: %w", err)
	}
	// 2. Load the yaml file into cfg
	var cfg config.Config
	if _, err := cfg.LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("could not load config %q: %w", configPath, err)
	}
	var databases []backup.Database
	// 3. Build Postgres instances
	for _, postgresInstance := range cfg.PostgresInstances {
		opts := []backup.PostgresOption{
			backup.WithPostgresCredentials(postgresInstance.Username, postgresInstance.Password),
			backup.WithPostgresDatabase(postgresInstance.Database),
			backup.WithPostgresHost(postgresInstance.Host),
			backup.WithPostgresPort(postgresInstance.Port),
			backup.WithPostgresMethod(postgresInstance.Method),
		}
		pg, err := backup.NewPostgres(cfg, opts...)
		pg.Logger = log
		if err != nil {
			return nil, fmt.Errorf(
				"failed creating Postgres instance for %q: %w",
				postgresInstance.Database,
				err,
			)
		}
		databases = append(databases, pg)
	}

	// 4) Build MongoDB instances
	for _, mongoInstance := range cfg.MongoInstances {
		opts := []backup.MongoDBOption{
			backup.WithMongoCredentials(mongoInstance.Username, mongoInstance.Password),
			backup.WithMongoDatabase(mongoInstance.Database),
			backup.WithMongoHost(mongoInstance.Host),
			backup.WithMongoPort(mongoInstance.Port),
			backup.WithMongoMethod(mongoInstance.Method),
		}

		mdb, err := backup.NewMongoDB(cfg, opts...)
		mdb.Logger = log
		if err != nil {
			return nil, fmt.Errorf(
				"failed creating MongoDB instance for %q: %w",
				mongoInstance.Database,
				err,
			)
		}
		databases = append(databases, mdb)
	}
	return databases, nil
}

// BackupDatabase runs a single backup against one Database.
// It returns an error if that backup fails.
func BackupDatabase(db backup.Database) error {
	if err := db.Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	return nil
}

// BackupAll loads all database instances from configPath and
// performs BackupDatabase on each in turn. On first error, it
// aborts and returns that error. Finally, it flushes the logger.
func BackupAll(configPath string) error {
	instances, err := InitializeDatabases(configPath)
	if err != nil {
		return err
	}

	for _, inst := range instances {
		if err := BackupDatabase(inst); err != nil {
			return err
		}
	}

	// Ensure buffered log entries are written out
	logger.Cleanup()
	return nil
}
