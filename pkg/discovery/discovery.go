package discovery

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"path/filepath"
	"resticontainer/pkg/apptypes"
	"strings"
)

func ParseContainer(container types.ContainerJSON, hostMountPrefix string) (*apptypes.Target, error) {
	labels := container.Config.Labels
	if labels["restic.enable"] != "true" {
		return nil, nil
	}

	target := &apptypes.Target{
		ContainerID: container.ID,
		PreHook:     labels["restic.hooks.pre-backup"],
		PostHook:    labels["restic.hooks.post-backup"],
	}

	pathsStr := labels["restic.backup.paths"]
	if pathsStr == "" {
		return target, nil
	}

	paths := strings.Split(pathsStr, ",")
	for _, p := range paths {
		p = strings.TrimSpace(p)
		found := false
		for _, m := range container.Mounts {
			if m.Destination == p {
				hostPath := filepath.Join(hostMountPrefix, m.Source)
				target.Paths = append(target.Paths, hostPath)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Warning: path %s in container %s is not a mounted volume, skipping\n", p, container.ID)
		}
	}

	return target, nil
}
