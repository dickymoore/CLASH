package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"clash/internal/contextinfo"
)

// Entry captures a single decision and execution attempt.
type Entry struct {
	ID               string                `json:"id"`
	Timestamp        time.Time             `json:"timestamp"`
	Command          string                `json:"command"`
	Cwd              string                `json:"cwd"`
	RepoRoot         string                `json:"repo_root"`
	Git              contextinfo.GitSummary `json:"git"`
	Decision         string                `json:"decision"`
	Hard             bool                  `json:"hard"`
	Signals          []string              `json:"signals"`
	Reasons          []string              `json:"reasons"`
	SaferAlternative string                `json:"safer_alternative"`
	Preview          *PreviewRecord        `json:"preview,omitempty"`
	ApprovedBy       string                `json:"approved_by,omitempty"`
	BreakGlass       bool                  `json:"break_glass"`
	BreakGlassReason string                `json:"break_glass_reason,omitempty"`
	Outcome          string                `json:"outcome"`
	ExitCode         int                   `json:"exit_code"`
	Error            string                `json:"error,omitempty"`
}

// PreviewRecord stores the preview summary.
type PreviewRecord struct {
	Count  int      `json:"count"`
	Sample []string `json:"sample"`
	Note   string   `json:"note"`
	Err    string   `json:"err,omitempty"`
}

// Logger writes audit events to disk.
type Logger struct {
	path string
}

// New creates a logger rooted at the repo or user home.
func New(repoRoot string) (*Logger, error) {
	path := ""
	if repoRoot != "" {
		path = filepath.Join(repoRoot, ".clash", "audit.log")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".clash", "audit.log")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Logger{path: path}, nil
}

// Record appends an audit entry as JSONL.
func (l *Logger) Record(entry Entry) error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(entry)
}

// Find returns the first entry matching the ID.
func (l *Logger) Find(id string) (Entry, error) {
	f, err := os.Open(l.path)
	if err != nil {
		return Entry{}, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.ID == id {
			return e, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return Entry{}, err
	}
	return Entry{}, errors.New("audit id not found")
}

// Path returns the path to the log file.
func (l *Logger) Path() string {
	return l.path
}
