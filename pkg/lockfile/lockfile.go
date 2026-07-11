package lockfile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const version = 1

// Entry is a single container's remembered backup state.
type Entry struct {
	Name      string    `json:"name,omitempty"`
	Paths     []string  `json:"paths"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LockFile records the resolved host paths backed up for each container, so those paths
// keep being backed up even when the container is no longer running.
type LockFile struct {
	Version    int              `json:"version"`
	Containers map[string]Entry `json:"containers"`
}

func empty() *LockFile {
	return &LockFile{Version: version, Containers: map[string]Entry{}}
}

// Load reads the lock file at path. A missing file yields an empty LockFile and a nil
// error. A corrupt or unreadable file yields an empty LockFile and a non-nil error, so
// the caller can warn and continue rather than lose a backup.
func Load(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return empty(), nil
		}
		return empty(), err
	}

	var l LockFile
	if err := json.Unmarshal(data, &l); err != nil {
		return empty(), err
	}
	if l.Containers == nil {
		l.Containers = map[string]Entry{}
	}
	if l.Version == 0 {
		l.Version = version
	}
	return &l, nil
}

// Upsert records or replaces a container's entry.
func (l *LockFile) Upsert(id, name string, paths []string, now time.Time) {
	l.Containers[id] = Entry{
		Name:      name,
		Paths:     paths,
		UpdatedAt: now,
	}
}

// Delete removes a container's entry. Deleting a missing id is a no-op.
func (l *LockFile) Delete(id string) {
	delete(l.Containers, id)
}

// BackupPaths returns the deduped, existence-filtered, stably sorted union of every
// entry's paths. exists reports whether a host path is still present on disk; paths for
// which it returns false are skipped. A nil exists keeps every path.
func (l *LockFile) BackupPaths(exists func(string) bool) []string {
	set := map[string]struct{}{}
	for _, e := range l.Containers {
		for _, p := range e.Paths {
			if exists == nil || exists(p) {
				set[p] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// Save writes the lock file atomically: it marshals to a temp file in the destination
// directory and renames it into place, creating the parent directory if needed.
func Save(path string, l *LockFile) error {
	if l.Version == 0 {
		l.Version = version
	}
	if l.Containers == nil {
		l.Containers = map[string]Entry{}
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".restic-backup-lock-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
