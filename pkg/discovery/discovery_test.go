package discovery

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/assert"
)

func TestParseContainer(t *testing.T) {
	cJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID: "12345",
		},
		Config: &container.Config{
			Labels: map[string]string{
				"restic.enable":            "true",
				"restic.backup.paths":      "/data,/etc/config,/missing_path",
				"restic.hooks.pre-backup":  "echo pre",
				"restic.hooks.post-backup": "echo post",
			},
		},
		Mounts: []types.MountPoint{
			{
				Destination: "/data",
				Source:      "/var/lib/docker/volumes/foo/_data",
				Type:        mount.TypeVolume,
			},
			{
				Destination: "/etc/config",
				Source:      "/host/etc/config",
				Type:        mount.TypeBind,
			},
			{
				Destination: "/ignored",
				Source:      "/ignored/path",
				Type:        mount.TypeBind,
			},
		},
	}

	target, err := ParseContainer(cJSON, "/hostfs")
	assert.NoError(t, err)
	assert.NotNil(t, target)
	assert.Equal(t, "12345", target.ContainerID)
	assert.Equal(t, "echo pre", target.PreHook)
	assert.Equal(t, "echo post", target.PostHook)
	assert.Equal(t, []string{"/hostfs/var/lib/docker/volumes/foo/_data", "/hostfs/host/etc/config"}, target.Paths)
}

func TestParseContainer_Subpaths(t *testing.T) {
	cJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{ID: "sub"},
		Config: &container.Config{
			Labels: map[string]string{
				"restic.enable": "true",
				// exact mount; a subdir of a mount; a subdir of the more-specific
				// nested mount; a false-prefix (skipped); an unmounted path (skipped).
				"restic.backup.paths": "/data,/data/library,/data/media/x,/database,/missing",
			},
		},
		Mounts: []types.MountPoint{
			{Destination: "/data", Source: "/host/data", Type: mount.TypeBind},
			{Destination: "/data/media", Source: "/host/media", Type: mount.TypeBind},
		},
	}
	target, err := ParseContainer(cJSON, "/hostfs")
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"/hostfs/host/data",         // /data (exact)
		"/hostfs/host/data/library", // /data/library (subdir of /data)
		"/hostfs/host/media/x",      // /data/media/x (subdir of the nested /data/media — longest match wins)
	}, target.Paths)
}

func TestParseContainer_NotEnabled(t *testing.T) {
	cJSON := types.ContainerJSON{
		Config: &container.Config{
			Labels: map[string]string{},
		},
	}
	target, err := ParseContainer(cJSON, "/hostfs")
	assert.NoError(t, err)
	assert.Nil(t, target)
}

func TestParseContainer_LockLabelAndName(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		want   bool
	}{
		{"absent defaults to true", map[string]string{"restic.enable": "true"}, true},
		{"explicit true", map[string]string{"restic.enable": "true", "restic.backup.lock": "true"}, true},
		{"explicit false disables", map[string]string{"restic.enable": "true", "restic.backup.lock": "false"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cJSON := types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{ID: "id", Name: "/myctr"},
				Config:            &container.Config{Labels: tc.labels},
			}
			target, err := ParseContainer(cJSON, "/hostfs")
			assert.NoError(t, err)
			assert.NotNil(t, target)
			assert.Equal(t, tc.want, target.Lock)
			assert.Equal(t, "myctr", target.Name) // leading slash trimmed
		})
	}
}
