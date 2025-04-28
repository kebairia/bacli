package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Backup            BackupConfig       `yaml:"backup"`
	Defaults          DefaultConfig      `yaml:"defaults"`
	PostgresInstances []PostgreSQLConfig `yaml:"postgres_instances"`
	MongoInstances    []MongoDBConfig    `yaml:"mongodb_instances"`
}
type BackupConfig struct {
	OutputDir       string `yaml:"output_dir"`
	Compress        bool   `yaml:"compress"`
	TimestampFormat string `yaml:"timestamp_format"`
}
type DefaultConfig struct {
	Postgres PostgresDefaults `yaml:"postgres"`
	MongoDB  MongoDefaults    `yaml:"mongodb"`
}
type PostgresDefaults struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}
type MongoDefaults struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

type PostgreSQLConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

type MongoDBConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Method   string `yaml:"method,omitempty"`
}

func (config *Config) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("unmarshal YAML config: %w", err)
	}
	// Validate backup section
	if config.Backup.OutputDir == "" {
		return fmt.Errorf("backup.output_dir is required")
	}
	// Provide default timestamp format if unset
	if config.Backup.TimestampFormat == "" {
		config.Backup.TimestampFormat = "2006-01-02_15-04-05"
	}
	// Validate Postgres instances
	for i, ps := range config.PostgresInstances {
		if ps.Database == "" {
			return fmt.Errorf("postgres_instances[%d].database is required", i)
		}
	}
	// Validate MongoDB instances
	for i, m := range config.MongoInstances {
		if m.Database == "" {
			return fmt.Errorf("mongodb_instances[%d].database is required", i)
		}
	}
	return nil
}
