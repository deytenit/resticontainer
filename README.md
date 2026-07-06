# resticontainer

A lightweight wrapper around [Restic](https://restic.net/) that adds automatic discovery and backup of Docker containers using OCI labels.

## Features

- **Transparent Pass-Through:** Any standard `restic` command (e.g. `init`, `snapshots`, `forget`) is passed directly to the underlying `restic` binary.
- **Docker Auto-Discovery:** When running `backup`, `resticontainer` queries the Docker daemon for active containers with specific labels.
- **Dynamic Path Resolution:** Translates container volume mounts back to the host filesystem.
- **Pre & Post Hooks:** Execute shell scripts inside the container immediately before and after the backup runs (e.g. `pg_dump`, `FLUSH TABLES`).

## Docker Labels

To enable a container to be backed up by `resticontainer`, add the following labels to your Docker container or Compose service:

- `restic.enable=true`: **(Required)** Enables backup for this container.
- `restic.backup.paths=/data,/etc/config`: **(Required)** A comma-separated list of container paths to back up. *These must correspond to active bind mounts or named volumes.*
- `restic.hooks.pre-backup`: *(Optional)* A command to run inside the container before the backup starts.
- `restic.hooks.post-backup`: *(Optional)* A command to run inside the container after the backup finishes (runs even if the backup fails).
- `restic.backup.stop=true`: *(Optional)* Automatically stops the container *after* running pre-hooks, performs the backup, and starts it again *before* running post-hooks. Great for ensuring database files (like SQLite) are cleanly flushed to disk. (You can also use `restic.backup.down=true`).

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
  -e RESTIC_HOSTFS=/hostfs \
  -e RESTIC_PASSWORD="your-password" \
  ghcr.io/deytenit/resticontainer:latest \
  backup --repo s3:s3.amazonaws.com/my-bucket
```

> **Note**: You must mount the Docker socket so `resticontainer` can query the API and execute hooks, and you must mount the host root (`/`) to `/hostfs:ro` so `resticontainer` can read the data from the host filesystem.

## Versioning

The `resticontainer` Docker image tags follow the upstream `restic` versions. For example, `ghcr.io/deytenit/resticontainer:0.16.4` contains `resticontainer` bundled with `restic v0.16.4`.

## Acknowledgments

This project is a thin orchestration layer standing on the shoulders of giants. A special thanks to the [Restic](https://github.com/restic/restic) team for creating such a fast, reliable, and secure backup program.

## License

This project is licensed under the BSD 2-Clause License, adhering to the same licensing terms as Restic itself. See the [LICENSE](LICENSE) file for details.
