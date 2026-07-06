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
