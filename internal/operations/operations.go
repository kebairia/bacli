package operations

import (
	"context"
	"fmt"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/database"
	"github.com/kebairia/backup/internal/logger"
	"github.com/kebairia/backup/internal/vault"
)

// OperationManager is a struct that manages the backup and restore operations.
type OperationManager struct {
	ctx         context.Context
	cfg         config.Config
	vaultClient *vault.Client
	log         logger.Logger
}

// NOTE: I don't see any use of the context here, check it

// NewOperationManager loads, parses, and validates the YAML config at configPath.
func NewOperationManager(configPath string) (*OperationManager, error) {
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

	return &OperationManager{
		ctx:         ctx,
		cfg:         config,
		vaultClient: vaultClient,
		log:         log,
	}, nil
}

// InitializeDatabases loads, parses, and validates the YAML config at configPath.
func (om *OperationManager) InitializeDatabases() ([]database.Database, error) {
	// pg_dbs, err := om.InitializePostgresInstance()
	pg_dbs, err := database.InitPostgresInstances(om.cfg, om.ctx, om.vaultClient)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres instance: %w", err)
	}
	mg_dbs, err := database.InitMongoDBInstances(om.cfg, om.ctx, om.vaultClient)
	if err != nil {
		return nil, fmt.Errorf("initialize mongodb instance: %w", err)
	}

	dbs := make([]database.Database, 0, len(pg_dbs)+len(mg_dbs))
	dbs = append(dbs, pg_dbs...)
	dbs = append(dbs, mg_dbs...)

	return dbs, nil
}
