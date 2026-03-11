package contract

import "time"

// RunEnvelope is the stable machine-readable execution contract.
type RunEnvelope struct {
	Task            string    `json:"task"`
	OK              bool      `json:"ok"`
	ExitCode        int       `json:"exit_code"`
	DurationMS      int64     `json:"duration_ms"`
	Stdout          string    `json:"stdout"`
	Stderr          string    `json:"stderr"`
	Artifacts       []string  `json:"artifacts"`
	StartedAt       time.Time `json:"started_at"`
	StdoutTruncated bool      `json:"stdout_truncated"`
	StderrTruncated bool      `json:"stderr_truncated"`
	StdoutBytes     int64     `json:"stdout_bytes"`
	StderrBytes     int64     `json:"stderr_bytes"`
}

// DryRunEnvelope describes what would execute when --dry-run is used.
type DryRunEnvelope struct {
	Task    string            `json:"task"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Cwd     string            `json:"cwd"`
	Timeout string            `json:"timeout"`
	Env     map[string]string `json:"env"`
}
