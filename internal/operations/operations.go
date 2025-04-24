package operations

import (
	"fmt"

	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
)

func InitializeInstances() (databases []backup.Database, err error) {
	var cfg config.Config
	if _, err := cfg.LoadConfig("./configs/config.yaml"); err != nil {
		return nil, fmt.Errorf("could not load config %w", err)
	}
	// Postgres instances
	for _, ps := range cfg.PostgresInstances {
		opts := []backup.Option{
			backup.WithUsername(ps.Username),
			backup.WithPassword(ps.Password),
			backup.WithDatabase(ps.Database),
			backup.WithHost(ps.Host),
			backup.WithPort(ps.Port),
			backup.WithMethod(ps.Method),
		}
		databases = append(databases, backup.NewPostgresInstance(cfg, opts...))
	}
	// MongoDB instances
	for _, mongo := range cfg.MongoInstances {

		opts := []backup.MongoOption{
			backup.MongoWithUsername(mongo.Username),
			backup.MongoWithPassword(mongo.Password),
			backup.MongoWithDatabase(mongo.Database),
			backup.MongoWithHost(mongo.Host),
			backup.MongoWithPort(mongo.Port),
			backup.MongoWithMethod(mongo.Method),
		}
		databases = append(databases, backup.NewMongoDBInstance(cfg, opts...))
	}
	return databases, nil
}

func PerformBackup(db backup.Database) error {
	logger.Init()

	if err := db.Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	return nil
}

func PerformAllBackups() error {
	databases, err := InitializeInstances()
	if err != nil {
		panic(err)
	}
	for _, db := range databases {
		if err := db.Backup(); err != nil {
			return fmt.Errorf("backup failed : %w", err)
		}
	}
	return nil
}
