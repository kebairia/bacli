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
)

// MongoDBOption lets you override default settings on a MongoDB.
type MongoDBOption func(*MongoDB)

// MongoDB holds configuration for backing up and restoring a MongoDB database.
type MongoDB struct {
	Username        string
	Password        string
	Database        string
	Host            string
	Port            string
	Method          string
	OutputDir       string
	TimeStampFormat string
	Logger          logger.Logger
}

// NewMongoDB returns a MongoDB configured from cfg plus any overrides.
// It also initializes the global logger.
func NewMongoDB(cfg config.Config, opts ...MongoDBOption) (*MongoDB, error) {
	log, err := logger.Init()
	if err != nil {
		return nil, fmt.Errorf("logger init failed: %w", err)
	}

	// Start with defaults from your YAML config
	m := &MongoDB{
		Username:        cfg.Defaults.MongoDB.Username,
		Password:        cfg.Defaults.MongoDB.Password,
		Database:        cfg.Defaults.MongoDB.Database,
		Host:            cfg.Defaults.MongoDB.Host,
		Port:            cfg.Defaults.MongoDB.Port,
		Method:          cfg.Defaults.MongoDB.Method,
		OutputDir:       cfg.Backup.OutputDir,
		TimeStampFormat: cfg.Backup.TimestampFormat,
		Logger:          log,
	}
	// Apply any functional options
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// WithMongoHost overrides the MongoDB host.
func WithMongoHost(host string) MongoDBOption {
	return func(m *MongoDB) {
		if host != "" {
			m.Host = host
		}
	}
}

// WithMongoPort overrides the MongoDB port.
func WithMongoPort(port string) MongoDBOption {
	return func(m *MongoDB) {
		if port != "" {
			m.Port = port
		}
	}
}

// WithMongoCredentials sets username and password.
func WithMongoCredentials(user, pass string) MongoDBOption {
	return func(m *MongoDB) {
		if user != "" {
			m.Username = user
		}
		if pass != "" {
			m.Password = pass
		}
	}
}

// WithMongoDatabase overrides the database name.
func WithMongoDatabase(db string) MongoDBOption {
	return func(m *MongoDB) {
		if db != "" {
			m.Database = db
		}
	}
}

// WithPostgresMethod overrides output format (custom/plain/directory).
func WithMongoMethod(method string) MongoDBOption {
	return func(m *MongoDB) {
		if method != "" {
			m.Method = method
		}
	}
}

// WithMongoOutputDir overrides where backups are written.
func WithMongoOutputDir(dir string) MongoDBOption {
	return func(m *MongoDB) {
		if dir != "" {
			m.OutputDir = dir
		}
	}
}

// WithTimestampFormat overrides timestamp format
func WithMongoTimestampFormat(timeStampFormat string) MongoDBOption {
	return func(m *MongoDB) {
		if timeStampFormat != "" {
			m.TimeStampFormat = timeStampFormat
		}
	}
}

// Backup runs `mongodump` to back up the database into a timestamped folder.
func (m *MongoDB) Backup() error {
	log := m.Logger
	// Build a path like ./backups/mongodb/2025-04-24_21-00-00-mydb
	timestamp := time.Now().Format(m.TimeStampFormat)
	outputPath := filepath.Join(m.OutputDir, "mongodb", fmt.Sprintf("%s-%s", timestamp, m.Database))

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", filepath.Dir(outputPath), err)
	}

	// Prepare arguments for mongodump
	args := []string{
		"--host=" + m.Host,
		"--port=" + m.Port,
		"--username=" + m.Username,
		"--password=" + m.Password,
		"--db=" + m.Database,
		"--authenticationDatabase=admin",
		"--out=" + outputPath,
	}

	cmd := exec.CommandContext(context.Background(), "mongodump", args...)
	// Silence mongodump’s stdout, keep stderr for errors
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	log.Info("Starting MongoDB backup",
		"database", m.Database,
		"output", outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongodump failed: %w", err)
	}

	log.Info("MongoDB backup complete",
		"database", m.Database,
		"output", outputPath,
	)
	return nil
}

// Restore runs `mongorestore` to restore from a backup directory.
func (m *MongoDB) Restore(sourceDir string) error {
	log := m.Logger
	// Ensure the source exists
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("backup source %q not found: %w", sourceDir, err)
	}

	args := []string{
		"--host=" + m.Host,
		"--port=" + m.Port,
		"--username=" + m.Username,
		"--password=" + m.Password,
		"--db=" + m.Database,
		"--authenticationDatabase=admin",
		"--drop", // drop existing collections first
		"--dir=" + sourceDir,
	}

	cmd := exec.CommandContext(context.Background(), "mongorestore", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	log.Info("Starting MongoDB restore",
		"database", m.Database,
		"source", sourceDir,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongorestore failed: %w", err)
	}

	log.Info("MongoDB restore complete",
		"database", m.Database,
	)
	return nil
}
