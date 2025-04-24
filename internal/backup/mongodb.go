package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
	"go.uber.org/zap"
)

type MongoDBInstance struct {
	Username   string
	Password   string
	DBName     string
	Host       string
	Port       string
	Method     string
	OutputPath string
}

// Default values "functional options pattern"
type MongoOption func(*MongoDBInstance)

// Option is a configuration option for PostgreSQLInstance.
func MongoWithHost(host string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.Host = host
	}
}

// WithPort overrides the port.
func MongoWithPort(port string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.Port = port
	}
}

// WithCompress override the compress
func MongoWithUsername(username string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.Username = username
	}
}

// WithOutputPath override the outputpath
func MongoWithOutputPath(outputPath string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.OutputPath = outputPath
	}
}

// WithPassword override the password
func MongoWithPassword(password string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.Password = password
	}
}

// WithDatabase override the database
func MongoWithDatabase(database string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.DBName = database
	}
}

// WithMethod override the database
func MongoWithMethod(method string) MongoOption {
	return func(ps *MongoDBInstance) {
		ps.Method = method
	}
}

// NewMongoDBInstance creates and returns a configured MongoDBInstance.
func NewMongoDBInstance(cfg config.Config, opts ...MongoOption) *MongoDBInstance {
	instance := &MongoDBInstance{
		Username:   cfg.Defaults.MongoDB.Username,
		Password:   cfg.Defaults.MongoDB.Password,
		DBName:     cfg.Defaults.MongoDB.Database,
		Host:       cfg.Defaults.MongoDB.Host,
		Port:       cfg.Defaults.MongoDB.Port,
		Method:     cfg.Defaults.MongoDB.Method,
		OutputPath: cfg.Backup.OutputDir,
	}

	for _, opt := range opts {
		opt(instance)
	}
	return instance
}

func (instance *MongoDBInstance) Backup() error {
	logger.Init()
	ctx := context.Background()
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	// I will add extension later
	backupFile := filepath.Join(
		instance.OutputPath,
		"mongodb",
		fmt.Sprintf("%s-%s.dump", timestamp, instance.DBName),
	)
	// Ensure the output directory exists
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	// Function to check if a file/directory exist or not.
	args := []string{
		"--host=" + instance.Host,
		"--port=" + instance.Port,
		"--username=" + instance.Username,
		"--password=" + instance.Password,
		"--db=" + instance.DBName,
		"--authenticationDatabase=admin",
		"--out=" + backupFile,
	}
	cmd := exec.CommandContext(ctx, "mongodump", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	logger.Log.Info("Starting MongoDB backup",
		zap.String("db", instance.DBName),
		zap.String("path", backupFile))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongodump failed: %w", err)
	}

	logger.Log.Info("MongoDB backup complete",
		zap.String("db", instance.DBName),
		zap.String("path", backupFile))

	return nil
}

func (instance *MongoDBInstance) Restore(filename string) error {
	logger.Init()
	// ctx := context.Background()

	// Check if the provided backup file exists
	if err := EnsureDirectoryExist(filename); err != nil {
		return err
	}
	return nil
}
