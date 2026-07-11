package lockfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Missing(t *testing.T) {
	lf, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	assert.NoError(t, err)
	assert.NotNil(t, lf)
	assert.Equal(t, 1, lf.Version)
	assert.Empty(t, lf.Containers)
}

func TestLoad_Corrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.json")
	assert.NoError(t, os.WriteFile(path, []byte("{not valid json"), 0o644))

	lf, err := Load(path)
	assert.Error(t, err)
	assert.NotNil(t, lf)
	assert.Empty(t, lf.Containers)
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	// Nested dir that does not exist yet — Save must create it.
	path := filepath.Join(t.TempDir(), "state", "restic-backup-lock.json")
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	lf := empty()
	lf.Upsert("c1", "pg", []string{"/hostfs/a", "/hostfs/b"}, now)
	assert.NoError(t, Save(path, lf))

	got, err := Load(path)
	assert.NoError(t, err)
	assert.Equal(t, lf.Version, got.Version)
	assert.Equal(t, lf.Containers, got.Containers)
}

func TestSave_AtomicLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "restic-backup-lock.json")

	lf := empty()
	lf.Upsert("c1", "pg", []string{"/x"}, time.Unix(0, 0).UTC())
	assert.NoError(t, Save(path, lf))

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "restic-backup-lock.json", entries[0].Name())
}

func TestUpsert_Replaces(t *testing.T) {
	lf := empty()
	now := time.Unix(0, 0).UTC()
	lf.Upsert("c1", "old", []string{"/a"}, now)
	lf.Upsert("c1", "new", []string{"/b"}, now)

	assert.Len(t, lf.Containers, 1)
	assert.Equal(t, "new", lf.Containers["c1"].Name)
	assert.Equal(t, []string{"/b"}, lf.Containers["c1"].Paths)
}

func TestDelete(t *testing.T) {
	lf := empty()
	lf.Upsert("c1", "", []string{"/a"}, time.Unix(0, 0).UTC())
	lf.Delete("c1")
	assert.Empty(t, lf.Containers)
	lf.Delete("missing") // must not panic
}

func TestBackupPaths_DedupFilterSort(t *testing.T) {
	lf := empty()
	now := time.Unix(0, 0).UTC()
	lf.Upsert("c1", "", []string{"/gone", "/b", "/a"}, now)
	lf.Upsert("c2", "", []string{"/a", "/c"}, now) // /a duplicated across containers

	exists := func(p string) bool { return p != "/gone" }
	assert.Equal(t, []string{"/a", "/b", "/c"}, lf.BackupPaths(exists))
}

func TestBackupPaths_NilExistsKeepsAll(t *testing.T) {
	lf := empty()
	lf.Upsert("c1", "", []string{"/b", "/a"}, time.Unix(0, 0).UTC())
	assert.Equal(t, []string{"/a", "/b"}, lf.BackupPaths(nil))
}
