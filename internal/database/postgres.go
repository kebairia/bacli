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

// PostgresOption lets you override default settings on a Postgres.
type PostgresOption func(*Postgres)

// Postgres holds configuration for backing up and restoring a PostgreSQL database.
type Postgres struct {
	Username        string
	Password        string
	Database        string
	Host            string
	Port            string
	Method          string // e.g. "custom", "plain", "directory"
	OutputDir       string
	TimeStampFormat string
	Timeout         time.Duration
	Logger          logger.Logger
}

// NewPostgres returns a Postgres configured from cfg plus any overrides.
// It also initializes the global logger.
func NewPostgres(cfg config.Config, opts ...PostgresOption) (*Postgres, error) {
	log, err := logger.Init()
	if err != nil {
		return nil, fmt.Errorf("logger init failed: %w", err)
	}
	p := &Postgres{
		Host:            cfg.Postgres.EngineDefaults.Host,
		Port:            cfg.Postgres.EngineDefaults.Port,
		Method:          cfg.Postgres.EngineDefaults.Method,
		OutputDir:       cfg.Backup.OutputDirectory,
		TimeStampFormat: cfg.Backup.TimestampFormat,
		Timeout:         cfg.Backup.Timeout,
		Logger:          log,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// WithPostgresHost overrides the host.
func WithPostgresHost(host string) PostgresOption {
	return func(p *Postgres) {
		if host != "" {
			p.Host = host
		}
	}
}

// WithPostgresPort overrides the port.
func WithPostgresPort(port string) PostgresOption {
	return func(p *Postgres) {
		if port != "" {
			p.Port = port
		}
	}
}

// WithPostgresCredentials sets username and password.
func WithPostgresCredentials(user, pass string) PostgresOption {
	return func(p *Postgres) {
		if user != "" {
			p.Username = user
		}
		if pass != "" {
			p.Password = pass
		}
	}
}

// WithPostgresDatabase overrides the database name.
func WithPostgresDatabase(db string) PostgresOption {
	return func(p *Postgres) {
		if db != "" {
			p.Database = db
		}
	}
}

// WithPostgresMethod overrides output format (custom/plain/directory).
func WithPostgresMethod(method string) PostgresOption {
	return func(p *Postgres) {
		if method != "" {
			p.Method = method
		}
	}
}

// WithPostgresOutputDir overrides where backups are written.
func WithPostgresOutputDir(dir string) PostgresOption {
	return func(p *Postgres) {
		if dir != "" {
			p.OutputDir = dir
		}
	}
}

// WithTimestampFormat overrides timestamp format
func WithPostgresTimestampFormat(timeStampFormat string) PostgresOption {
	return func(p *Postgres) {
		if timeStampFormat != "" {
			p.TimeStampFormat = timeStampFormat
		}
	}
}

// Backup runs `pg_dump` to back up the database into a timestamped .dump file.
func (p *Postgres) Backup() (backupPath string, err error) {
	log := p.Logger
	ctx, cancel := context.WithTimeoutCause(context.Background(), p.Timeout, ErrTimeout)

	defer cancel()
	// e.g. "./backups/postgres/2025-04-24_21-00-00-mydb.dump"
	timestamp := time.Now().Format(p.TimeStampFormat)
	backupPath = filepath.Join(
		p.OutputDir,
		"postgres",
		p.Database,
		fmt.Sprintf("%s-%s.dump", timestamp, p.Database),
	)

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", filepath.Dir(backupPath), err)
	}

	// Build pg_dump args
	args := []string{
		"-h", p.Host,
		"-p", p.Port,
		"-U", p.Username,
		"-d", p.Database,
		"-F", p.Method,
		"-f", backupPath,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	// Pass PGPASSWORD for non-interactive auth
	cmd.Env = append(os.Environ(), "PGPASSWORD="+p.Password)
	cmd.Stderr = os.Stderr

	log.Info("backup started",
		"database", p.Database,
		"engine", "postgres",
		"method", p.Method,
		"path", backupPath,
	)

	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pg_dump failed: %w", err)
	}
	executionDuration := time.Since(startTime)

	log.Info("backup completed",
		"database", p.Database,
		"engine", "postgres",
		"path", backupPath,
		"duration", executionDuration.String(),
	)
	return backupPath, nil
}

// Restore runs `pg_restore` to restore from a .dump file.
func (p *Postgres) Restore(backupFile string) error {
	log := p.Logger
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	// Ensure the file exists
	// Check if the backup source file exists
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup file %q not found: %w", backupFile, err)
	}

	// Build the right command based on p.Method
	var cmd *exec.Cmd
	switch p.Method {
	case "plain":
		// Plain SQL → use psql -f
		cmd = exec.CommandContext(ctx, "psql",
			"-h", p.Host,
			"-p", p.Port,
			"-U", p.Username,
			"-d", p.Database,
			"-f", backupFile,
		)

	case "custom", "directory", "tar":
		cmd = exec.CommandContext(ctx, "pg_restore",
			"-h", p.Host,
			"-p", p.Port,
			"-U", p.Username,
			"-d", p.Database,
			"-c", // Clean existing objects
			"-F", p.Method,
			backupFile,
		)
	default:
		return fmt.Errorf("unsupported restore method %q", p.Method)
	}

	// Handle non interactive authorization
	cmd.Env = append(os.Environ(), "PGPASSWORD="+p.Password)
	cmd.Stdout = io.Discard // I don't want to see the restoring output of postgres
	cmd.Stderr = os.Stderr

	// Logging

	log.Info("restore started",
		"database", p.Database,
		"engine", "postgres",
		"method", p.Method,
		"source", backupFile,
	)

	// Run and check for errors
	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restore failed (%s): %w", p.Method, err)
	}
	executionDuration := time.Since(startTime)

	log.Info("restore completed",
		"database", p.Database,
		"engine", "postgres",
		"source", backupFile,
		"duration", executionDuration.String(),
	)
	return nil
}

func (p *Postgres) GetName() string {
	return p.Database
}

func (p *Postgres) GetEngine() string {
	return "postgres"
}

func (p *Postgres) GetPath() string {
	return filepath.Join(p.OutputDir, "postgres")
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
		secrets, err := vaultClient.GetDynamicCredentials(ctx, rolePath)
		if err != nil {
			return nil, fmt.Errorf("vault read :%w", err)
		}
		opts := []PostgresOption{
			WithPostgresHost(instance.Host),
			WithPostgresPort(instance.Port),
			WithPostgresCredentials(secrets.Username, secrets.Password),
			WithPostgresDatabase(instance.Database),
			WithPostgresMethod(instance.Method),
			WithPostgresOutputDir(cfg.Backup.OutputDirectory),
			WithPostgresTimestampFormat(cfg.Backup.TimestampFormat),
		}
		db, err := NewPostgres(cfg, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize postgres instance: %w", err)
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}
