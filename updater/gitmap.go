package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GitMapRelease represents a .gitmap/release/*.json entry.
type GitMapRelease struct {
	Version      string `json:"version"`
	Branch       string `json:"branch"`
	SourceBranch string `json:"sourceBranch"`
	Tag          string `json:"tag"`
	IsLatest     bool   `json:"isLatest"`
}

// gitmapDir is the relative path to the gitmap release directory.
const gitmapDir = ".gitmap/release"

// readGitMapLatest reads .gitmap/release/latest.json from the given repo root.
func readGitMapLatest(repoRoot string) (*GitMapRelease, error) {
	path := filepath.Join(repoRoot, gitmapDir, "latest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rel GitMapRelease
	if err := json.Unmarshal(data, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}
