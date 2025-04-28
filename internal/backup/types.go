package backup

import "time"

// Metadata for a single DB backup run
type DBRecord struct {
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration_ms"`
	SizeBytes int64         `json:"size_bytes"`
}

// Top-level metadata file
type Metadata struct {
	RunAt   time.Time  `json:"run_at"`
	Backups []DBRecord `json:"backups"`
}
