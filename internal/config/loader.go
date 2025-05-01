package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the entire backup configuration loaded from YAML files.
type Config struct {
	Include           []string           `yaml:"include"`
	Backup            BackupConfig       `yaml:"backup"`
	Metadata          MetadataConfig     `yaml:"metadata"`
	Defaults          DefaultConfig      `yaml:"defaults"`
	PostgresInstances []PostgreSQLConfig `yaml:"postgres_instances"`
	MongoInstances    []MongoDBConfig    `yaml:"mongodb_instances"`
}

// BackupConfig defines global backup settings.
type BackupConfig struct {
	OutputDir       string        `yaml:"output_dir"`
	Compress        bool          `yaml:"compress"`
	TimestampFormat string        `yaml:"timestamp_format"`
	Timeout         time.Duration `yaml:"timeout"`
}

// MetadataConfig defines where metadata.json is stored.
type MetadataConfig struct {
	Path string `yaml:"path"`
}

// DefaultConfig holds default connection settings for all database types.
type DefaultConfig struct {
	Postgres PostgresDefaults `yaml:"postgres"`
	MongoDB  MongoDefaults    `yaml:"mongodb"`
}

// PostgresDefaults are fallback settings for Postgres instances.
type PostgresDefaults struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

// MongoDefaults are fallback settings for MongoDB instances.
type MongoDefaults struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

// PostgreSQLConfig defines a single Postgres instance configuration.
type PostgreSQLConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

// MongoDBConfig defines a single MongoDB instance configuration.
type MongoDBConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

// IncludedInstances is used to parse postgres/mongo instances from included files.
type IncludedInstances struct {
	PostgresInstances []PostgreSQLConfig `yaml:"postgres_instances"`
	MongoInstances    []MongoDBConfig    `yaml:"mongodb_instances"`
}

// LoadConfig loads the main configuration file and all included instance files.
func (config *Config) LoadConfig(filename string) error {
	// 1. Load the main configuration file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read main config file: %w", err)
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse main config file: %w", err)
	}

	// 2. Load included instance files, if any
	for _, includePath := range config.Include {
		if err := config.loadIncludedFile(includePath); err != nil {
			return fmt.Errorf("failed to load included file %q: %w", includePath, err)
		}
	}

	return nil
}

// loadIncludedFile loads postgres.yaml, mongodb.yaml, etc., and appends instances to the config.
func (config *Config) loadIncludedFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read included file %q: %w", path, err)
	}

	var included IncludedInstances
	if err := yaml.Unmarshal(data, &included); err != nil {
		return fmt.Errorf("failed to parse included file %q: %w", path, err)
	}

	// Append found instances
	if len(included.PostgresInstances) > 0 {
		config.PostgresInstances = append(config.PostgresInstances, included.PostgresInstances...)
	}
	if len(included.MongoInstances) > 0 {
		config.MongoInstances = append(config.MongoInstances, included.MongoInstances...)
	}

	return nil
}
