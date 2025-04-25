package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/logger"
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
		Username:        cfg.Defaults.Postgres.Username,
		Password:        cfg.Defaults.Postgres.Password,
		Database:        cfg.Defaults.Postgres.Database,
		Host:            cfg.Defaults.Postgres.Host,
		Port:            cfg.Defaults.Postgres.Port,
		Method:          cfg.Defaults.Postgres.Method,
		OutputDir:       cfg.Backup.OutputDir,
		TimeStampFormat: cfg.Backup.TimestampFormat,
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
func (p *Postgres) Backup() error {
	log := p.Logger
	// e.g. "./backups/postgres/2025-04-24_21-00-00-mydb.dump"
	timestamp := time.Now().Format(p.TimeStampFormat)
	backupFile := filepath.Join(
		p.OutputDir,
		"postgres",
		fmt.Sprintf("%s-%s.dump", timestamp, p.Database),
	)

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(backupFile), 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", filepath.Dir(backupFile), err)
	}

	// Build pg_dump args
	args := []string{
		"-h", p.Host,
		"-p", p.Port,
		"-U", p.Username,
		"-d", p.Database,
		"-F", p.Method,
		"-f", backupFile,
	}

	cmd := exec.CommandContext(context.Background(), "pg_dump", args...)
	// Pass PGPASSWORD for non-interactive auth
	cmd.Env = append(os.Environ(), "PGPASSWORD="+p.Password)
	cmd.Stderr = os.Stderr

	log.Info("Starting PostgreSQL backup",
		"database", p.Database,
		"method", p.Method,
		"output", backupFile,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	log.Info("PostgreSQL backup complete",
		"database", p.Database,
		"output", backupFile,
	)
	return nil
}

// Restore runs `pg_restore` to restore from a .dump file.
func (p *Postgres) Restore(backupFile string) error {
	log := p.Logger
	// Ensure the file exists
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup file %q not found: %w", backupFile, err)
	}

	args := []string{
		"-h", p.Host,
		"-p", p.Port,
		"-U", p.Username,
		"-d", p.Database,
		"-c",           // clean: drop objects before recreate
		"-F", p.Method, // must match how it was dumped
		backupFile,
	}

	cmd := exec.CommandContext(context.Background(), "pg_restore", args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+p.Password)
	cmd.Stderr = os.Stderr

	log.Info("Starting PostgreSQL restore",
		"database", p.Database,
		"method", p.Method,
		"file", backupFile,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	log.Info("PostgreSQL restore complete",
		"database", p.Database,
	)
	return nil
}
