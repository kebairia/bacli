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
)

const mysqlEngine = "mysql"

// MySQLOption lets you override default settings on a MySQL.
type MySQLOption func(*MySQL)

// MySQL holds configuration for backing up and restoring a MySQL database.
type MySQL struct {
	Username        string
	Password        string
	Database        string
	Host            string
	Port            string
	Method          string // e.g. "dump"
	OutputDir       string
	TimeStampFormat string
	Timeout         time.Duration
	Logger          logger.Logger
}

// NewMySQL returns a MySQL configured from cfg plus any overrides.
func NewMySQL(cfg config.Config, opts ...MySQLOption) (*MySQL, error) {
	log, err := logger.Init()
	if err != nil {
		return nil, fmt.Errorf("logger init failed: %w", err)
	}
	m := &MySQL{
		Host:            cfg.MySQL.EngineDefaults.Host,
		Port:            cfg.MySQL.EngineDefaults.Port,
		Method:          cfg.MySQL.EngineDefaults.Method,
		OutputDir:       cfg.Backup.OutputDirectory,
		TimeStampFormat: cfg.Backup.TimestampFormat,
		Timeout:         cfg.Backup.Timeout,
		Logger:          log,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// WithMySQLCredentials sets username and password.
func WithMySQLCredentials(user, pass string) MySQLOption {
	return func(m *MySQL) {
		if user != "" {
			m.Username = user
		}
		if pass != "" {
			m.Password = pass
		}
	}
}

// WithMySQLHost overrides the host.
func WithMySQLHost(host string) MySQLOption {
	return func(m *MySQL) {
		if host != "" {
			m.Host = host
		}
	}
}

// WithMySQLPort overrides the port.
func WithMySQLPort(port string) MySQLOption {
	return func(m *MySQL) {
		if port != "" {
			m.Port = port
		}
	}
}

// WithMySQLDatabase sets the database name.
func WithMySQLDatabase(db string) MySQLOption {
	return func(m *MySQL) {
		if db != "" {
			m.Database = db
		}
	}
}

// WithMySQLOutputDir overrides where backups are written.
func WithMySQLOutputDir(dir string) MySQLOption {
	return func(m *MySQL) {
		if dir != "" {
			m.OutputDir = dir
		}
	}
}

// WithMySQLTimestampFormat overrides timestamp format.
func WithMySQLTimestampFormat(format string) MySQLOption {
	return func(m *MySQL) {
		if format != "" {
			m.TimeStampFormat = format
		}
	}
}

// Backup runs `mysqldump` to back up the database into a timestamped .sql file.
func (m *MySQL) Backup() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	fileName := fmt.Sprintf("%s-%s.sql", time.Now().Format(m.TimeStampFormat), m.Database)
	backupsDir := filepath.Join(m.OutputDir, mysqlEngine, m.Database)
	backupPath := filepath.Join(backupsDir, fileName)

	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", backupsDir, err)
	}

	// Build mysqldump args
	args := []string{
		"-h", m.Host,
		"-P", m.Port,
		"-u", m.Username,
		"--databases", m.Database,
		"--single-transaction",
		"--result-file=" + backupPath,
	}
	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	// Pass MYSQL_PWD for non-interactive auth
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+m.Password)
	cmd.Stderr = os.Stderr

	m.Logger.Info("backup started",
		"database", m.Database,
		"engine", mysqlEngine,
		"path", backupPath,
	)
	start := time.Now()
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mysqldump failed: %w", err)
	}
	m.Logger.Info("backup completed", "duration", time.Since(start).String())

	return backupPath, nil
}

// Restore runs `mysql` to restore from a .sql file.
func (m *MySQL) Restore(backupFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	// Ensure file exists
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup file %q not found: %w", backupFile, err)
	}

	cmd := exec.CommandContext(ctx, "mysql",
		"-h", m.Host,
		"-P", m.Port,
		"-u", m.Username,
		m.Database,
	)
	// Pass MYSQL_PWD for non-interactive auth
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+m.Password)

	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer file.Close()
	cmd.Stdin = file
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	m.Logger.Info("restore started", "database", m.Database, "engine", mysqlEngine)
	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w", err)
	}
	m.Logger.Info("restore completed", "duration", time.Since(start).String())
	return nil
}

// GetName returns database name.
func (m *MySQL) GetName() string { return m.Database }

// GetEngine returns engine name.
func (m *MySQL) GetEngine() string { return mysqlEngine }

// GetPath returns the base backup path.
func (m *MySQL) GetPath() string { return filepath.Join(m.OutputDir, mysqlEngine) }
