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

// OperationManager is a struct that manages the backup and restore operations.
type OperationManager struct {
	cfg         config.Config
	vaultClient *vault.Client
	log         logger.Logger
}

// NewOperationManager loads, parses, and validates the YAML config at configPath.
func NewOperationManager(configPath string) (*OperationManager, error) {
	var config config.Config
	if err := config.Load(configPath); err != nil {
		return nil, err
	}
	// Init Vault client
	vaultClient, err := vault.NewClient(config.Vault.Address, config.Vault.Token)
	if err != nil {
		return nil, fmt.Errorf("vault client init: %w", err)
	}

	log := logger.Global()

	return &OperationManager{
		cfg:         config,
		vaultClient: vaultClient,
		log:         log,
	}, nil
}

// InitializeMongoDBInstance loads, parses, and validates the YAML config at configPath.
func (om *OperationManager) InitializeMongoDBInstance() ([]backup.Database, error) {
	var dbs []backup.Database
	for _, instance := range om.cfg.MongoDB.Instances {
		secretPath := filepath.Join(om.cfg.MongoDB.Vault.KVBase, instance.KVPath)
		rolePath := filepath.Join(om.cfg.MongoDB.Vault.RoleBase, instance.RoleName)
		connection, err := om.vaultClient.FullDBConnection(secretPath, rolePath)
		if err != nil {
			return nil, fmt.Errorf("vault read : %w", err)
		}
		opts := []backup.MongoDBOption{
			backup.WithMongoCredentials(connection.Username, connection.Password),
			backup.WithMongoDatabase(connection.Database),
			backup.WithMongoHost(connection.Host),
			backup.WithMongoPort(connection.Port),
			backup.WithMongoMethod(instance.Method),
		}
		mg, err := backup.NewMongoDB(om.cfg, opts...)
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
func (om *OperationManager) InitializePostgresInstance() ([]backup.Database, error) {
	var dbs []backup.Database
	for _, instance := range om.cfg.Postgres.Instances {
		secretPath := filepath.Join(om.cfg.Postgres.Vault.KVBase, instance.KVPath)
		rolePath := filepath.Join(om.cfg.Postgres.Vault.RoleBase, instance.RoleName)
		connection, err := om.vaultClient.FullDBConnection(secretPath, rolePath)
		if err != nil {
			return nil, fmt.Errorf("vault read :%w", err)
		}
		opts := []backup.PostgresOption{
			backup.WithPostgresCredentials(connection.Username, connection.Password),
			backup.WithPostgresDatabase(connection.Database),
			backup.WithPostgresHost(connection.Host),
			backup.WithPostgresPort(connection.Port),
			backup.WithPostgresMethod(instance.Method),
		}
		pg, err := backup.NewPostgres(om.cfg, opts...)
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
func (om *OperationManager) InitializeDatabases() ([]backup.Database, error) {
	pg_dbs, err := om.InitializePostgresInstance()
	if err != nil {
		return nil, fmt.Errorf("initialize postgres instance: %w", err)
	}
	mg_dbs, err := om.InitializeMongoDBInstance()
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
func (om *OperationManager) BackupDatabase(
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

// RestoreDatabase runs a single restore against one Database.
// It returns an error if that restore fails.
// NOTE: Check for metadata.json in the backup directory,
// NOTE: if not exist, return an error
// FIX: I should remove the RestoreDatabase() and the BackupDatabase() methods form
// operation manager
func (om *OperationManager) RestoreDatabase(db backup.Database, record backup.Metadata) error {
	// decompress the file if the compress=true
	if om.cfg.Backup.Compress {
		decPath, err := DecompressZstd(record.FilePath)
		if err != nil {
			return err
		}
		record.FilePath = decPath

		// Remove the decompressed files
		defer backup.RemoveFile(record.FilePath)
	}
	if err := db.Restore(record.FilePath); err != nil {
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

		wg.Add(1)
		// start of the goroutine
		go func(db backup.Database) {
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

	record := backup.Metadata{}
	var wg sync.WaitGroup

	for _, db := range databases {
		wg.Add(1)

		go func(db backup.Database, record backup.Metadata) {
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
