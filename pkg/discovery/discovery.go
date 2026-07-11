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
		Name:        strings.TrimPrefix(container.Name, "/"),
		PreHook:     labels["restic.hooks.pre-backup"],
		PostHook:    labels["restic.hooks.post-backup"],
		Stop:        labels["restic.backup.stop"] == "true" || labels["restic.backup.down"] == "true",
		Lock:        labels["restic.backup.lock"] != "false",
	}

	pathsStr := labels["restic.backup.paths"]
	if pathsStr == "" {
		return target, nil
	}

	for _, p := range strings.Split(pathsStr, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if hostPath, ok := resolveHostPath(container.Mounts, p, hostMountPrefix); ok {
			target.Paths = append(target.Paths, hostPath)
		} else {
			fmt.Printf("Warning: path %s in container %s is not under a mounted volume, skipping\n", p, container.ID)
		}
	}

	return target, nil
}

// resolveHostPath maps a container path to its path on the host. The path may be a
// mount destination itself, or a subdirectory of one — so a single mount (e.g. /data)
// can be backed up selectively (e.g. /data/library) without a separate mount. The
// longest matching mount destination wins, so nested mounts resolve to the most
// specific source.
func resolveHostPath(mounts []types.MountPoint, p, prefix string) (string, bool) {
	best := -1
	var hostPath string
	for _, m := range mounts {
		rel, ok := underMount(p, m.Destination)
		if ok && len(m.Destination) > best {
			best = len(m.Destination)
			hostPath = filepath.Join(prefix, m.Source, rel)
		}
	}
	return hostPath, best >= 0
}

// underMount reports whether container path p equals mount destination dst or lives
// under it, returning p relative to dst ("" when they are equal). The trailing
// separator guards against false prefixes (e.g. /database is not under /data).
func underMount(p, dst string) (string, bool) {
	if p == dst {
		return "", true
	}
	d := dst
	if d != "/" {
		d += "/"
	}
	if strings.HasPrefix(p, d) {
		return strings.TrimPrefix(p, d), true
	}
	return "", false
}
