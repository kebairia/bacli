package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// ErrLoadConfig indicates a failure to read or parse the YAML configuration.
var ErrLoadConfig = errors.New("config load failed")

// ErrValidateConfig indicates that the loaded configuration is invalid.
var ErrValidateConfig = errors.New("configuration validation failed")

// Config represents the top-level YAML configuration file.
type Config struct {
	Include   []string        `mapstructure:"include"   yaml:"include,omitempty"`
	Backup    BackupConfig    `mapstructure:"backup"    yaml:"backup"`
	Vault     VaultConfig     `mapstructure:"vault"     yaml:"vault"`
	Retention RetentionConfig `mapstructure:"retention" yaml:"retention"`

	// Per-engine groups
	Postgres DBGroupConfig `mapstructure:"postgres" yaml:"postgres"`
	MongoDB  DBGroupConfig `mapstructure:"mongodb"  yaml:"mongodb"`
	MySQL    DBGroupConfig `mapstructure:"mysql"    yaml:"mysql"`
	Redis    DBGroupConfig `mapstructure:"redis"    yaml:"redis"`
}

// VaultConfig holds connection settings for HashiCorp Vault.
type VaultConfig struct {
	Address     string `mapstructure:"address"      yaml:"address"`
	ApproleName string `mapstructure:"approle_name" yaml:"approle_name,omitempty"`
}

// BackupConfig contains global backup options.
type BackupConfig struct {
	OutputDirectory string        `mapstructure:"output_directory" yaml:"output_directory"`
	Compress        bool          `mapstructure:"compress"         yaml:"compress"`
	TimestampFormat string        `mapstructure:"timestamp_format" yaml:"timestamp_format"`
	Timeout         time.Duration `mapstructure:"timeout"          yaml:"timeout"`
}

// RetentionConfig specifies how many backups to keep and cleanup interval.
type RetentionConfig struct {
	KeepLast        int           `mapstructure:"keep_last"        yaml:"keep_last"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval" yaml:"cleanup_interval"`
}

// EngineDefaults provides common settings for a DB engine.
type EngineDefaults struct {
	Host     string        `mapstructure:"host"     yaml:"host,omitempty"`
	Port     string        `mapstructure:"port"     yaml:"port,omitempty"`
	Timeout  time.Duration `mapstructure:"timeout"  yaml:"timeout,omitempty"`
	Compress bool          `mapstructure:"compress" yaml:"compress,omitempty"`
	Method   string        `mapstructure:"method"   yaml:"method,omitempty"`
}

// DBGroupConfig groups common engine settings and Vault prefixes.
type DBGroupConfig struct {
	EngineDefaults `mapstructure:",squash" yaml:",inline"` // inline and squash host, port, timeout, compress, method

	Vault     VaultPaths   `mapstructure:"vault"     yaml:"vault"`
	Instances []DBInstance `mapstructure:"instances" yaml:"instances"`
}

// VaultPaths holds the KV and role prefixes under the Vault mount.
type VaultPaths struct {
	KVBase   string `mapstructure:"kv_base"   yaml:"kv_base"`
	RoleBase string `mapstructure:"role_base" yaml:"role_base"`
}

// DBInstance represents a single database within a group.
type DBInstance struct {
	Name     string `mapstructure:"name"      yaml:"name"`
	Host     string `mapstructure:"host"      yaml:"host,omitempty"`
	Port     string `mapstructure:"port"      yaml:"port,omitempty"`
	Database string `mapstructure:"database"  yaml:"database,omitempty"`
	RoleName string `mapstructure:"role_name" yaml:"role_name,omitempty"`
	Method   string `mapstructure:"method"    yaml:"method,omitempty"`
	Compress bool   `mapstructure:"compress"  yaml:"compress,omitempty"`
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
