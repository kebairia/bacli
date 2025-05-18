package operations

import (
	"context"
	"fmt"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/database"
	"github.com/kebairia/backup/internal/logger"
	"github.com/kebairia/backup/internal/vault"
)

// Operator manages the lifecycle of database backup and restore operations.
// and other !operations!
// It holds the execution context, configuration, Vault client, and logger.
type Operator struct {
	ctx         context.Context
	config      config.Config
	vaultClient *vault.Client
	log         logger.Logger
}

// Operator methods:
//
// InitializeDatabases prepares and configures the target databases before any
// backup or restore action.
//
// Backup creates a snapshot of the databases and writes it to the configured
// storage backend.
//
// Restore retrieves a backup snapshot and restores it to the target databases.
//
// Cleanup removes temporary files and releases resources used during backup or
// restore.
//
// Validate checks the integrity and consistency of database files and backups.
//
// Compress reduces the size of backup files using the configured compression
// algorithm.
//
// Encrypt secures backup files with the configured encryption settings.
//
// Decrypt makes encrypted backups readable by removing encryption layers.

// NewOperator creates a new Operator with the given context, configuration,
// Vault client, and logger.

func NewOperator(configPath string) (*Operator, error) {
	var config config.Config
	ctx := context.Background()
	if err := config.Load(configPath); err != nil {
		return nil, err
	}
	//  Build Vault options
	vaultOpts := []vault.Option{
		vault.WithAddress(config.Vault.Address),
		vault.WithAppRole(config.Vault.RoleID, config.Vault.RoleName),
	}
	// Init Vault client
	vaultClient, err := vault.NewClient(ctx, vaultOpts...)
	if err != nil {
		return nil, fmt.Errorf("vault client init: %w", err)
	}

	log := logger.Global()

	return &Operator{
		ctx:         ctx,
		config:      config,
		vaultClient: vaultClient,
		log:         log,
	}, nil
}

// NOTE: I can use a map
//
//	instances := make(map[string][]database.Database)
//
// InitializeDatabases loads, parses, and validates the YAML config at configPath.
func (operator *Operator) InitializeDatabases(engines []string) ([]database.Database, error) {
	var err error
	instances := make(map[string][]database.Database)
	dbs := make([]database.Database, 0)

	for _, engine := range engines {
		switch engine {
		case "postgres":
			instances[engine], err = database.InitPostgresInstances(
				operator.config,
				operator.ctx,
				operator.vaultClient,
			)
			if err != nil {
				return nil, fmt.Errorf("initialize postgres instance: %w", err)
			}
			dbs = append(dbs, instances[engine]...)
		case "mongodb":
			instances[engine], err = database.InitMongoDBInstances(
				operator.config,
				operator.ctx,
				operator.vaultClient,
			)
			if err != nil {
				return nil, fmt.Errorf("initialize mongodb instance: %w", err)
			}
			dbs = append(dbs, instances[engine]...)
			// case "mysql":
			// 	instances[engine], err = database.InitMySQLInstances(operator.config, operator.ctx, operator.vaultClient)
			// 	if err != nil {
			// 		return nil, fmt.Errorf("initialize mysql instance: %w", err)
			// 	}
			// 	dbs = append(dbs, instances[engine]...)
			// case "redis":
			// 	instances[engine], err = database.InitRedisInstances(operator.config, operator.ctx, operator.vaultClient)
			// 	if err != nil {
			// 		return nil, fmt.Errorf("initialize redis instance: %w", err)
			// 	}
			//
			// 	dbs = append(dbs, instances[engine]...)
		}
	}

	return dbs, nil
}
