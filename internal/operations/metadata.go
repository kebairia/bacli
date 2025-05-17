package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const MetadataFilename = "metadata.json"

// Metadata for a single DB backup run
type Metadata struct {
	Engine      string        `json:"engine"`
	Database    string        `json:"database"`
	FilePath    string        `json:"file_path"`
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration_ms"`
	SizeBytes   int64         `json:"size_bytes"`
}

// metadata file
func (m *Metadata) Load(filePath string) error {
	// Build full path to metadata file
	dirPath := filepath.Dir(filePath)

	// Ensure directory exists
	if err := EnsureDirectoryExist(dirPath); err != nil {
		return fmt.Errorf("metadata directory %q not found: %w", dirPath, err)
	}
	// Open the json file
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("couldn't open metadata file %q: %w", filePath, err)
	}
	defer jsonFile.Close()
	// Decode JSON directory from file into the Metadata struct
	decoder := json.NewDecoder(jsonFile)
	if err := decoder.Decode(m); err != nil {
		return fmt.Errorf("decode metadata JSON: %w", err)
	}
	return nil
}

// Write metadata file
func (m *Metadata) Write(dirPath string) error {
	// Build full path to metadata file
	filePath := filepath.Join(dirPath, MetadataFilename)

	// Ensure directory exists
	if err := EnsureDirectoryExist(dirPath); err != nil {
		return fmt.Errorf("ensure metadata directory %q: %w", dirPath, err)
	}

	// Create the json file
	jsonFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create metadata file %q: %w", filePath, err)
	}
	defer jsonFile.Close()

	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(m); err != nil {
		return fmt.Errorf("encode metadata JSON: %w", err)
	}
	return nil
}
