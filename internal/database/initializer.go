package database

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/vault"
)

var engines = []string{"postgres", "mongodb", "mysql", "redis"}

var initializers = map[string]func(
	cfg config.Config,
	ctx context.Context,
	vaultClient *vault.Client,
) ([]Database, error){
	"postgres": InitPostgresInstances,
	"mongodb":  InitMongoDBInstances,
	// "mysql":    InitMySQLInstances,
	// "redis":    InitRedisInstances,
}

// InitializePostgresInstance loads, parses, and validates the YAML config at configPath.
func InitPostgresInstances(
	cfg config.Config,
	ctx context.Context,
	vaultClient *vault.Client,
) ([]Database, error) {
	var dbs []Database
	for _, instance := range cfg.Postgres.Instances {
		rolePath := filepath.Join(cfg.Postgres.Vault.RoleBase, instance.RoleName)
		creds, err := vaultClient.GetDynamicCredentials(ctx, rolePath)
		if err != nil {
			return nil, fmt.Errorf("vault read :%w", err)
		}
		opts := []PostgresOption{
			WithPostgresHost(instance.Host),
			WithPostgresPort(instance.Port),
			WithPostgresCredentials(creds.Username, creds.Password),
			WithPostgresDatabase(instance.Database),
			WithPostgresMethod(instance.Method),
			WithPostgresOutputDir(cfg.Backup.OutputDirectory),
			WithPostgresTimestampFormat(cfg.Backup.TimestampFormat),
			WithPostgresCompress(true),
		}
		db, err := NewPostgres(cfg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize postgres instance: %w", err)
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

// InitMongoDBInstance loads, parses, and validates the YAML config at configPath.
func InitMongoDBInstances(
	cfg config.Config,
	ctx context.Context,
	vaultClient *vault.Client,
) ([]Database, error) {
	var dbs []Database
	for _, instance := range cfg.MongoDB.Instances {
		rolePath := filepath.Join(cfg.MongoDB.Vault.RoleBase, instance.RoleName)
		secrets, err := vaultClient.GetDynamicCredentials(ctx, rolePath)
		if err != nil {
			return nil, fmt.Errorf("vault read :%w", err)
		}
		opts := []MongoDBOption{
			WithMongoHost(instance.Host),
			WithMongoPort(instance.Port),
			WithMongoCredentials(secrets.Username, secrets.Password),
			WithMongoDatabase(instance.Database),
			WithMongoMethod(instance.Method),
			WithMongoOutputDir(cfg.Backup.OutputDirectory),
			WithMongoTimestampFormat(cfg.Backup.TimestampFormat),
		}
		db, err := NewMongoDB(cfg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize mongodb instance: %w", err)
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

// initMySQLInstances initializes MySQL instances.
// func initMySQLInstances(
// 	cfg config.Config,
// 	vaultClient *vault.Client,
// 	ctx context.Context,
// ) ([]Database, error) {
// 	var dbs []Database
// 	for _, inst := range cfg.MySQL.Instances {
// 		rolePath := filepath.Join(cfg.MySQL.Vault.RoleBase, inst.RoleName)
// 		creds, err := vaultClient.GetDynamicCredentials(ctx, rolePath)
// 		if err != nil {
// 			return nil, fmt.Errorf("vault read for mysql %q: %w", inst.Name, err)
// 		}
//
// 		opts := []MySQLOption{
// 			WithMySQLCredentials(creds.Username, creds.Password),
// 			WithMySQLHost(inst.Host),
// 			WithMySQLPort(inst.Port),
// 			WithMySQLDatabase(inst.Database),
// 			WithMySQLMethod(inst.Method),
// 			WithMySQLOutputDir(cfg.Backup.OutputDirectory),
// 			WithMySQLTimestampFormat(cfg.Backup.TimestampFormat),
// 		}
//
// 		my, err := NewMySQL(cfg, opts...)
// 		if err != nil {
// 			return nil, fmt.Errorf("create mysql instance %q: %w", inst.Name, err)
// 		}
// 		dbs = append(dbs, my)
// 	}
// 	return dbs, nil
// }
//
// // initRedisInstances initializes Redis instances.
// func initRedisInstances(
// 	cfg config.Config,
// 	vaultClient *vault.Client,
// 	ctx context.Context,
// ) ([]Database, error) {
// 	var dbs []Database
// 	for _, inst := range cfg.Redis.Instances {
// 		rolePath := filepath.Join(cfg.Redis.Vault.RoleBase, inst.RoleName)
// 		creds, err := vaultClient.GetDynamicCredentials(ctx, rolePath)
// 		if err != nil {
// 			return nil, fmt.Errorf("vault read for redis %q: %w", inst.Name, err)
// 		}
//
// 		opts := []RedisOption{
// 			WithRedisCredentials(creds.Username, creds.Password),
// 			WithRedisHost(inst.Host),
// 			WithRedisPort(inst.Port),
// 			WithRedisDatabase(inst.Database),
// 			WithRedisMethod(inst.Method),
// 			WithRedisOutputDir(cfg.Backup.OutputDirectory),
// 			WithRedisTimestampFormat(cfg.Backup.TimestampFormat),
// 		}
//
// 		rd, err := backup.NewRedis(cfg, opts...)
// 		if err != nil {
// 			return nil, fmt.Errorf("create redis instance %q: %w", inst.Name, err)
// 		}
// 		dbs = append(dbs, rd)
// 	}
// 	return dbs, nil
// }

// InitializeDatabases loads, parses, and validates the YAML config at configPath.
func InitializeDatabases(
	config config.Config,
	vaultClient *vault.Client,
	ctx context.Context,
) ([]Database, error) {
	dbs := make([]Database, 0)

	// NOTE: I can add `if !config.IsEngineEnabled(engine) { continue }` in the initializers
	// 			 to check first if the engine is enabled, I need to see if this is necessary or not.
	for engine, initializer := range initializers {
		instances, err := initializer(config, ctx, vaultClient)
		if err != nil {
			return nil, fmt.Errorf("initialize %s instance: %w", engine, err)
		}
		dbs = append(dbs, instances...)
	}

	return dbs, nil
}
