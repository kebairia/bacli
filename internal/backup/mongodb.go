package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kebairia/backup/internal/logger"
	"go.uber.org/zap"
)

type MongoDBInstance struct {
	Username   string
	Password   string
	DBName     string
	Host       string
	Port       string
	OutputPath string
}

func (instance *MongoDBInstance) Backup() error {
	logger.Init()
	ctx := context.Background()
	timestamp := time.Now().Format("2006-01-02_14-04-05")
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

// NewMongoDBInstance creates and returns a configured MongoDBInstance.
func NewMongoDBInstance(
	username, password, dbname, host, port, outputPath string,
) *MongoDBInstance {
	return &MongoDBInstance{
		Username:   username,
		Password:   password,
		DBName:     dbname,
		Host:       host,
		Port:       port,
		OutputPath: outputPath,
	}
}
