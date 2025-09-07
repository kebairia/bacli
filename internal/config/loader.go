package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

// ErrLoadConfig indicates a failure to read or parse the YAML configuration.
var ErrLoadConfig = errors.New("config load failed")

// ErrValidateConfig indicates that the loaded configuration is invalid.
var ErrValidateConfig = errors.New("configuration validation failed")

// -----------------------------------------------------------------------------
// Root Config
// -----------------------------------------------------------------------------

// Config represents the top-level YAML configuration file.
type Config struct {
	Include   []string        `mapstructure:"include"   yaml:"include,omitempty"`
	Vault     VaultConfig     `mapstructure:"vault"     yaml:"vault"`
	Backup    BackupConfig    `mapstructure:"backup"    yaml:"backup"`
	Retention RetentionConfig `mapstructure:"retention" yaml:"retention"`

	// Per-engine groups
	Postgres DBGroupConfig `mapstructure:"postgres" yaml:"postgres"`
	MongoDB  DBGroupConfig `mapstructure:"mongodb"  yaml:"mongodb"`
	MySQL    DBGroupConfig `mapstructure:"mysql"    yaml:"mysql"`
	Redis    DBGroupConfig `mapstructure:"redis"    yaml:"redis"`
}

// -----------------------------------------------------------------------------
// Vault
// -----------------------------------------------------------------------------

// VaultConfig holds connection settings for HashiCorp Vault.
type VaultConfig struct {
	Address string `mapstructure:"address" yaml:"address"`
	Approle string `mapstructure:"approle" yaml:"approle,omitempty"`
}

// VaultPaths holds the Vault path prefixes for DB credentials.
type VaultPaths struct {
	CredsPath string `mapstructure:"creds_path" yaml:"creds_path"`
}

// -----------------------------------------------------------------------------
// Backup
// -----------------------------------------------------------------------------

// BackupConfig contains global backup options.
type BackupConfig struct {
	Directory    string        `mapstructure:"directory"     yaml:"directory"`
	Compression  bool          `mapstructure:"compression"   yaml:"compression"`
	TimestampFmt string        `mapstructure:"timestamp_fmt" yaml:"timestamp_fmt"`
	Timeout      time.Duration `mapstructure:"timeout"       yaml:"timeout"`
}

// -----------------------------------------------------------------------------
// Retention
// -----------------------------------------------------------------------------

// RetentionConfig specifies how many backups to keep and cleanup interval.
type RetentionConfig struct {
	Keep     int           `mapstructure:"keep"     yaml:"keep"`
	Interval time.Duration `mapstructure:"interval" yaml:"interval"`
}

// -----------------------------------------------------------------------------
// Database Configs
// -----------------------------------------------------------------------------

// EngineDefaults provides common settings for a DB engine.
type EngineDefaults struct {
	Host        string        `mapstructure:"host"        yaml:"host,omitempty"`
	Port        string        `mapstructure:"port"        yaml:"port,omitempty"`
	Timeout     time.Duration `mapstructure:"timeout"     yaml:"timeout,omitempty"`
	Role        string        `mapstructure:"role"        yaml:"role,omitempty"`
	Compression bool          `mapstructure:"compression" yaml:"compression,omitempty"`
	Format      string        `mapstructure:"format"      yaml:"format,omitempty"`
	Method      string        `mapstructure:"format"      yaml:"format,omitempty"`
}

// DBGroupConfig groups engine-level defaults and Vault prefixes.
type DBGroupConfig struct {
	EngineDefaults `mapstructure:",squash" yaml:",inline"` // inline default fields

	Vault     VaultPaths   `mapstructure:"vault"     yaml:"vault"`
	Instances []DBInstance `mapstructure:"instances" yaml:"instances"`
}

// DBInstance represents a single database within a group.
type DBInstance struct {
	Name        string `mapstructure:"name"        yaml:"name"`
	Host        string `mapstructure:"host"        yaml:"host,omitempty"`
	Port        string `mapstructure:"port"        yaml:"port,omitempty"`
	Database    string `mapstructure:"database"    yaml:"database,omitempty"`
	Role        string `mapstructure:"role"        yaml:"role,omitempty"`
	Format      string `mapstructure:"format"      yaml:"format,omitempty"`
	Compression bool   `mapstructure:"compression" yaml:"compression,omitempty"`
	Method      string `mapstructure:"format"      yaml:"format,omitempty"`
}

// Load reads the configuration from the given YAML file using Viper,
// merges any included files, and unmarshals into the Config struct.
func (c *Config) Load(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetConfigType("yml")
	v.AutomaticEnv()

	// Read base configuration
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("%w: read base config %s: %v", ErrLoadConfig, path, err)
	}

	// Merge include files (if any)
	for _, inc := range v.GetStringSlice("include") {
		data, err := os.ReadFile(inc)
		if err != nil {
			return fmt.Errorf("%w: read include %s: %v", ErrLoadConfig, inc, err)
		}
		if err := v.MergeConfig(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("%w: merge include %s: %v", ErrLoadConfig, inc, err)
		}
	}

	// Unmarshal into the Config struct
	if err := v.UnmarshalExact(c); err != nil {
		return fmt.Errorf("%w: unmarshal config: %v", ErrLoadConfig, err)
	}

	return nil
}
