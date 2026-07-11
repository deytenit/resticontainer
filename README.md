# resticontainer

A lightweight wrapper around [Restic](https://restic.net/) that adds automatic discovery and backup of Docker containers using OCI labels.

## Features

- **Transparent Pass-Through:** Any standard `restic` command (e.g. `init`, `snapshots`, `forget`) is passed directly to the underlying `restic` binary.
- **Docker Auto-Discovery:** When running `backup`, `resticontainer` queries the Docker daemon for active containers with specific labels.
- **Dynamic Path Resolution:** Translates container volume mounts back to the host filesystem.
- **Pre & Post Hooks:** Execute shell scripts inside the container immediately before and after the backup runs (e.g. `pg_dump`, `FLUSH TABLES`).
- **Down-Container Resilience:** Remembers the paths it has backed up in a lock file, so they keep being backed up even when a container is later stopped or removed.

## Docker Labels

To enable a container to be backed up by `resticontainer`, add the following labels to your Docker container or Compose service:

- `restic.enable=true`: **(Required)** Enables backup for this container.
- `restic.backup.paths=/data,/etc/config`: **(Required)** A comma-separated list of container paths to back up. Each path must be an active bind mount / named volume **or a subdirectory of one** — so a single mount like `/data` can be backed up selectively (e.g. `restic.backup.paths=/data/library,/data/upload`, leaving regenerable siblings like `/data/thumbs` out). The most specific (longest-matching) mount is used to resolve the host path.
- `restic.hooks.pre-backup`: *(Optional)* A command to run inside the container before the backup starts.
- `restic.hooks.post-backup`: *(Optional)* A command to run inside the container after the backup finishes (runs even if the backup fails).
- `restic.backup.stop=true`: *(Optional)* Automatically stops the container *after* running pre-hooks, performs the backup, and starts it again *before* running post-hooks. Great for ensuring database files (like SQLite) are cleanly flushed to disk. (You can also use `restic.backup.down=true`).
- `restic.backup.lock=false`: *(Optional, default `true`)* When enabled (the default), this container's resolved paths are recorded in the lock file so they keep being backed up even while the container is stopped or removed. Set it to `false` to opt out — the container is then backed up **only while running**, and any existing lock entry for it is dropped.

## Example: PostgreSQL Backup

```yaml
services:
  postgres:
    image: postgres:15
    volumes:
      - pgdata:/var/lib/postgresql/data
      - pgdumps:/dumps
    labels:
      - "restic.enable=true"
      # Back up the /dumps volume
      - "restic.backup.paths=/dumps"
      # Run a pg_dump to /dumps right before the backup
      - "restic.hooks.pre-backup=pg_dump -U postgres dbname > /dumps/db.sql"
      # Clean up the dump afterwards to save space
      - "restic.hooks.post-backup=rm /dumps/db.sql"
```

## Running the Backup

Run `resticontainer backup` just as you would run `restic backup`, providing your repository and password:

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /:/hostfs:ro \
  -v resticontainer_state:/var/lib/resticontainer \
  -e RESTIC_HOSTFS=/hostfs \
  -e RESTIC_PASSWORD="your-password" \
  ghcr.io/deytenit/resticontainer:latest \
  backup --repo s3:s3.amazonaws.com/my-bucket
```

> **Note**: You must mount the Docker socket so `resticontainer` can query the API and execute hooks, and you must mount the host root (`/`) to `/hostfs:ro` so `resticontainer` can read the data from the host filesystem.

## The Backup Lock File

Docker auto-discovery only sees *running* containers, so a container that is stopped or removed at backup time would otherwise be silently skipped. To prevent gaps, `resticontainer` records each backed-up container's resolved host paths in a lock file and, on every run, also backs up any remembered paths whose owning container is no longer running.

- **Location:** defaults to `/var/lib/resticontainer/restic-backup-lock.json`; override with the `RESTICONTAINER_LOCK` environment variable.
- **Persistence:** mount a writable volume at that directory (as in the example above) so the lock file survives container restarts.
- **Skip-missing:** remembered paths that no longer exist on disk are skipped, so removing a volume cleanly drops it from future backups.
- **Opt out:** set `restic.backup.lock=false` on a container to keep it out of the lock file.

## Versioning

The `resticontainer` Docker image tags follow the upstream `restic` versions. For example, `ghcr.io/deytenit/resticontainer:0.16.4` contains `resticontainer` bundled with `restic v0.16.4`.

## Acknowledgments

This project is a thin orchestration layer standing on the shoulders of giants. A special thanks to the [Restic](https://github.com/restic/restic) team for creating such a fast, reliable, and secure backup program.

## License

This project is licensed under the BSD 2-Clause License, adhering to the same licensing terms as Restic itself. See the [LICENSE](LICENSE) file for details.

---

*Portions of this project's source code may have been developed with the assistance of AI code-generation tools. Contributions made with the help of such tools are welcome, provided the code quality stays within an acceptable range and the contributor fully understands the submission they are making.*
