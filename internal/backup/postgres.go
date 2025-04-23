package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kebairia/backup/internal/logger"
	"go.uber.org/zap"
)

type PostgreSQLInstance struct {
	Username   string
	Password   string
	DBName     string
	Host       string
	Port       string
	OutputPath string
}

func (instance *PostgreSQLInstance) Backup() error {
	logger.Init()
	ctx := context.Background()
	timestamp := time.Now().Format("2006-01-02_14-04-05")
	backupFile := filepath.Join(
		instance.OutputPath,
		"postgres",
		fmt.Sprintf("%s-%s.dump", timestamp, instance.DBName),
	)
	// Ensure the output directory exists
	if err := EnsureDirectoryExist(backupFile); err != nil {
		return err
	}

	// Prepare pg_dump command with environment variable for password
	args := []string{
		"-h", instance.Host,
		"-p", instance.Port,
		"-U", instance.Username,
		"-d", instance.DBName,
		"-f", backupFile,
	}
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", instance.Password))
	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Log.Info("Starting PostgreSQL backup",
		zap.String("db", instance.DBName),
		zap.String("path", backupFile))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}
	logger.Log.Info("PostgreSQL backup complete: ",
		zap.String("db", instance.DBName),
		zap.String("path", backupFile))

	return nil
}

func (instance *PostgreSQLInstance) Restore(filename string) error {
	logger.Init()
	ctx := context.Background()

	// Check if the provided backup file exists
	if err := EnsureDirectoryExist(filename); err != nil {
		return err
	}

	// Prepare pg_restore command arguments
	args := []string{
		"-h", instance.Host, // PostgreSQL host
		"-p", instance.Port, // PostgreSQL port
		"-U", instance.Username, // PostgreSQL user
		"-d", instance.DBName, // Target database to restore into
		"--clean",     // Drop database objects before recreating them
		"--if-exists", // Only drop objects if they exist
		filename,      // Path to the backup file (custom or directory format)
	}

	// Create the pg_restore command with the given context and args
	cmd := exec.CommandContext(ctx, "pg_restore", args...)

	// Add password to environment variables for non-interactive authentication
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", instance.Password))

	// Forward stderr to the terminal for visibility
	cmd.Stderr = os.Stderr

	// Log start of restore
	logger.Log.Info("Starting PostgreSQL restore",
		zap.String("db", instance.DBName),
		zap.String("backup_file", filename),
	)

	// Run the pg_restore command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	// Log success
	logger.Log.Info("PostgreSQL restore completed successfully",
		zap.String("db", instance.DBName),
		zap.String("backup_file", filename),
	)

	return nil
}

// NewPostgresInstance creates and returns a configured PostgreSQLInstance.
func NewPostgresInstance(
	username, password, dbname, host, port, outputPath string,
) *PostgreSQLInstance {
	return &PostgreSQLInstance{
		Username:   username,
		Password:   password,
		DBName:     dbname,
		Host:       host,
		Port:       port,
		OutputPath: outputPath,
	}
}
