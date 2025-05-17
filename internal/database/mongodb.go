package database

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
	"github.com/kebairia/backup/internal/vault"
)

const (
	MethodDir         = "directory"
	MethodDirGzip     = "directory-gzip"
	MethodArchive     = "archive"
	MethodArchiveGzip = "archive-gzip"
)

// MongoDBOption defines a functional option for configuring a MongoDB instance.
type MongoDBOption func(*MongoDB)

// MongoDB represents a MongoDB database instance for backup and restore.
type MongoDB struct {
	Username        string
	Password        string
	Database        string
	Host            string
	Port            string
	Method          string
	OutputDir       string
	TimestampFormat string
	Timeout         time.Duration
	Logger          logger.Logger
}

// NewMongoDB creates a new MongoDB instance based on config defaults and supplied options.
func NewMongoDB(cfg config.Config, opts ...MongoDBOption) (*MongoDB, error) {
	log, err := logger.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	m := &MongoDB{
		// Username:        cfg.Defaults.MongoDB.Username,
		Host:            cfg.MongoDB.EngineDefaults.Host,
		Port:            cfg.MongoDB.EngineDefaults.Port,
		Method:          cfg.MongoDB.EngineDefaults.Method,
		OutputDir:       cfg.Backup.OutputDirectory,
		TimestampFormat: cfg.Backup.TimestampFormat,
		Timeout:         cfg.Backup.Timeout,
		Logger:          log,
	}

	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// WithMongoHost overrides the database host.
func WithMongoHost(host string) MongoDBOption {
	return func(m *MongoDB) {
		if host != "" {
			m.Host = host
		}
	}
}

// WithMongoPort overrides the database port.
func WithMongoPort(port string) MongoDBOption {
	return func(m *MongoDB) {
		if port != "" {
			m.Port = port
		}
	}
}

// WithMongoCredentials overrides the username and password.
func WithMongoCredentials(username, password string) MongoDBOption {
	return func(m *MongoDB) {
		if username != "" {
			m.Username = username
		}
		if password != "" {
			m.Password = password
		}
	}
}

// WithMongoDatabase overrides the database name.
func WithMongoDatabase(database string) MongoDBOption {
	return func(m *MongoDB) {
		if database != "" {
			m.Database = database
		}
	}
}

// WithMongoMethod overrides the dump/restore method.
func WithMongoMethod(method string) MongoDBOption {
	return func(m *MongoDB) {
		if method != "" {
			m.Method = method
		}
	}
}

// WithMongoOutputDir overrides the output directory.
func WithMongoOutputDir(dir string) MongoDBOption {
	return func(m *MongoDB) {
		if dir != "" {
			m.OutputDir = dir
		}
	}
}

// WithMongoTimestampFormat overrides the timestamp format.
func WithMongoTimestampFormat(format string) MongoDBOption {
	return func(m *MongoDB) {
		if format != "" {
			m.TimestampFormat = format
		}
	}
}

// Backup creates a backup of the MongoDB database using mongodump.
func (m *MongoDB) Backup() (backupPath string, err error) {
	log := m.Logger
	ctx, cancel := context.WithTimeoutCause(context.Background(), m.Timeout, ErrTimeout)
	defer cancel()

	timestamp := time.Now().Format(m.TimestampFormat)
	backupPath = filepath.Join(
		m.OutputDir,
		"mongodb",
		m.Database,
		fmt.Sprintf("%s-%s.dump", timestamp, m.Database),
	)

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	var args []string

	base := []string{
		"--host=" + m.Host,
		"--port=" + m.Port,
		"--username=" + m.Username,
		"--password=" + m.Password,
		"--authenticationDatabase=admin",
		"--db=" + m.Database,
		"--quiet",
	}
	switch m.Method {
	case MethodDir:
		args = append(base,
			"--out="+backupPath,
		)
	case MethodDirGzip:
		args = append(base,
			"--out="+backupPath,
			"--gzip",
		)

	case MethodArchive:
		args = append(base,
			"--archive="+backupPath,
		)
	case MethodArchiveGzip:
		args = append(base,
			"--archive="+backupPath,
			"--gzip",
		)

	}

	cmd := exec.CommandContext(ctx, "mongodump", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	log.Info("backup started",
		"database", m.Database,
		"engine", "mongodb",
		"method", m.Method,
		"path", backupPath,
	)
	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		log.Error("backup failed",
			"database", m.Database,
			"engine", "mongodb",
			"path", backupPath,
			"error", err.Error(),
		)
		return "", fmt.Errorf("mongodump failed: %w", err)
	}
	executionDuration := time.Since(startTime)

	log.Info("backup completed",
		"database", m.Database,
		"engine", "mongodb",
		"path", backupPath,
		"duration", executionDuration.String(),
	)
	return backupPath, nil
}

// Restore restores a MongoDB database from a backup directory using mongorestore.
func (m *MongoDB) Restore(sourceDir string) error {
	log := m.Logger
	ctx, cancel := context.WithTimeoutCause(context.Background(), m.Timeout, ErrTimeout)
	defer cancel()

	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("backup source %q not found: %w", sourceDir, err)
	}

	// NOTE: Add other options "--dir=" + sourceDir,
	var cmd *exec.Cmd
	base := []string{
		"--host=" + m.Host,
		"--port=" + m.Port,
		"--username=" + m.Username,
		"--password=" + m.Password,
		"--authenticationDatabase=admin",
		"--nsInclude=" + m.Database + ".*", // restore only this DB’s namespaces
		"--drop",                           // replace collections if they already exist
		"--quiet",
	}
	var args []string
	switch m.Method {
	case MethodDir:
		args = append(base,
			"--dir="+sourceDir,
		)

	case MethodDirGzip:
		args = append(base,
			"--dir="+sourceDir,
			"--gzip",
		)
	case MethodArchive:
		args = append(base,
			"--archive="+sourceDir, // read .archive file
		)
	case MethodArchiveGzip:
		args = append(base,
			"--archive="+sourceDir,
			"--gzip",
		)

	}

	cmd = exec.CommandContext(ctx, "mongorestore", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	log.Info("restore started",
		"database", m.Database,
		"engine", "mongodb",
		"method", m.Method,
		"source", sourceDir,
	)
	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongorestore failed: %w", err)
	}
	executionDuration := time.Since(startTime)

	log.Info("restore completed",
		"database", m.Database,
		"engine", "mongodb",
		"source", sourceDir,
		"duration", executionDuration.String(),
	)
	return nil
}

func (m *MongoDB) GetName() string {
	return m.Database
}

func (m *MongoDB) GetEngine() string {
	return "mongodb"
}

func (m *MongoDB) GetPath() string {
	return filepath.Join(m.OutputDir, "mongodb")
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
