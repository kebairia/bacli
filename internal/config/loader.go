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

func (config *Config) LoadConfig(filename string) (data []byte, err error) {
	data, err = os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return data, fmt.Errorf("failed to unmarshal yaml config: %w", err)
	}
	return data, nil
}
