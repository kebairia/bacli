package config

import (
	"os"
	"testing"
)

func TestLoadConfig_ParsesBackupTimeout(t *testing.T) {
	yaml := `
backup:
  output_dir: "/tmp/backups"
  compress: true
  timestamp_format: "2006-01-02_15-04-05"
defaults:
  postgres:
    host: "db.example.com"
    port: "5432"
    method: "plain"
  mongodb:
    host: "mongo.example.com"
    port: "27017"
`
	// Write it to a temp file
	tmp, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(yaml); err != nil {
		t.Fatalf("failed to write YAML: %v", err)
	}
	tmp.Close()
	// Load it
	var cfg Config
	err = cfg.LoadConfig(tmp.Name())
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
}
