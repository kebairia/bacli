package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// VaultConfig holds connection settings for HashiCorp Vault.
type VaultConfig struct {
	Address string `yaml:"address"`
	Token   string `yaml:"token,omitempty"`
}

// DBInstance represents a single database instance.
type DBInstance struct {
	Name   string `yaml:"name"`
	Method string `yaml:"method,omitempty"`
}

// DBGroup groups related database instances under the same Vault KV path.
type DBGroup struct {
	// Default dump/restore method for instances in this group (e.g. custom, plain).
	Method string `yaml:"method,omitempty"`
	// Prefix under the Vault KV engine, e.g. "secret/data/backups/postgres".
	VaultBasePath string `yaml:"vault_base_path"`
	VaultRolePath string `yaml:"vault_role_path"`
	// List of concrete database instances.
	Instances []DBInstance `yaml:"instances"`
}

// BackupConfig contains global backup parameters.
type BackupConfig struct {
	OutputDir       string        `yaml:"output_dir"`
	Compress        bool          `yaml:"compress"`
	TimestampFormat string        `yaml:"timestamp_format"`
	Timeout         time.Duration `yaml:"timeout"`
}

// MetadataConfig points to the metadata.json location.
type MetadataConfig struct {
	Path string `yaml:"path"`
}

// DefaultConfig provides per‑engine fallback values.
type DefaultConfig struct {
	Postgres struct {
		Host   string `yaml:"host,omitempty"`
		Port   string `yaml:"port,omitempty"`
		Method string `yaml:"method,omitempty"`
	} `yaml:"postgres"`
	Mongo struct {
		Host   string `yaml:"host,omitempty"`
		Port   string `yaml:"port,omitempty"`
		Method string `yaml:"method,omitempty"`
	} `yaml:"mongodb"`
}

// Config is the top‑level structure representing the entire YAML file.
type Config struct {
	Include  []string       `yaml:"include"`
	Vault    VaultConfig    `yaml:"vault"`
	Backup   BackupConfig   `yaml:"backup"`
	Metadata MetadataConfig `yaml:"metadata"`
	Defaults DefaultConfig  `yaml:"defaults"`

	Postgres DBGroup `yaml:"postgres"`
	Mongo    DBGroup `yaml:"mongodb"`
}

// Load reads the main YAML file and any referenced include files, merging the
// result into *Config.
func (config *Config) Load(filename string) error {
	// Load the main configuration file
	if err := config.parseFile(filename); err != nil {
		return fmt.Errorf("failed to load main config file: %w", err)
	}

	// Load included instance files, if any
	for _, file := range config.Include {
		if err := config.includeFile(file); err != nil {
			return fmt.Errorf("failed to load included file %q: %w", file, err)
		}
	}

	return nil
}

// includedFile loads postgres.yaml, mongodb.yaml, etc., and appends instances to the config.
func (config *Config) includeFile(path string) error {
	// 1. Load the main configuration file
	if err := config.parseFile(path); err != nil {
		return fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	// Temporary holder to detect which block is present
	var include struct {
		Postgres *DBGroup `yaml:"postgres"`
		Mongo    *DBGroup `yaml:"mongodb"`
	}

	// Merge Postgres group: override base path if provided and append instances
	if include.Postgres != nil {
		if include.Postgres.VaultBasePath != "" {
			config.Postgres.VaultBasePath = include.Postgres.VaultBasePath
		}
		config.Postgres.Instances = append(config.Postgres.Instances, include.Postgres.Instances...)
	}
	// Merge MongoDB group
	if include.Mongo != nil {
		if include.Mongo.VaultBasePath != "" {
			config.Mongo.VaultBasePath = include.Mongo.VaultBasePath
		}
		config.Mongo.Instances = append(config.Mongo.Instances, include.Mongo.Instances...)
	}

	return nil
}

// parseFile reads a YAML file from disk and unmarshals it into dst.
func (config *Config) parseFile(path string) error {
	// Loadaa the file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config file %q: %w", path, err)
	}
	return nil
}
