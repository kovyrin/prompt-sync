package conflict

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Issue represents a conflict or drift issue
type Issue struct {
	Type       string // "duplicate" or "drift"
	Path       string
	Details    string
	IsCritical bool // If true, should fail in strict mode
}

// Detector scans for conflicts and drift in rendered outputs
type Detector struct {
	strictMode bool
}

// New creates a new conflict detector
func New(strictMode bool) *Detector {
	return &Detector{strictMode: strictMode}
}

// ScanDirectory scans a directory for duplicate basenames and returns issues
func (d *Detector) ScanDirectory(dir string) ([]Issue, error) {
	var issues []Issue

	// Track basenames to detect duplicates
	basenames := make(map[string][]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		basename := filepath.Base(path)
		relPath, _ := filepath.Rel(dir, path)
		basenames[basename] = append(basenames[basename], relPath)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Check for duplicates
	for basename, paths := range basenames {
		if len(paths) > 1 {
			issue := Issue{
				Type:       "duplicate",
				Path:       basename,
				Details:    fmt.Sprintf("duplicate basename found in: %s", strings.Join(paths, ", ")),
				IsCritical: true, // Duplicates are always critical
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// CheckDrift compares file hashes against expected hashes from lock file
func (d *Detector) CheckDrift(files map[string]string) ([]Issue, error) {
	var issues []Issue

	for path, expectedHash := range files {
		actualHash, err := d.calculateFileHash(path)
		if err != nil {
			if os.IsNotExist(err) {
				issue := Issue{
					Type:       "drift",
					Path:       path,
					Details:    "file missing",
					IsCritical: true,
				}
				issues = append(issues, issue)
				continue
			}
			return nil, fmt.Errorf("failed to calculate hash for %s: %w", path, err)
		}

		if actualHash != expectedHash {
			issue := Issue{
				Type:       "drift",
				Path:       path,
				Details:    fmt.Sprintf("hash mismatch: expected %s, got %s", expectedHash, actualHash),
				IsCritical: true,
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// FilterCritical returns only critical issues if in strict mode
func (d *Detector) FilterCritical(issues []Issue) []Issue {
	if !d.strictMode {
		return issues
	}

	var critical []Issue
	for _, issue := range issues {
		if issue.IsCritical {
			critical = append(critical, issue)
		}
	}
	return critical
}

func (d *Detector) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
