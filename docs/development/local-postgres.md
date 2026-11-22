# Local PostgreSQL Development

Local development runs PostgreSQL in Docker and needs no Neon
credentials. Remote work (Fly development, production) is covered in
`fly-development.md` and `../operations/fly-production.md`.

## Image selection (verified 2026-11-22)

- postgresql.org lists PostgreSQL 19 as the current supported GA
  release (19.0, released 2026-11-19).
- Docker Hub `postgres:19`, `postgres:19.0`, and `postgres:latest`
  resolve to one digest; `19rc1`/`19beta1` resolve to different
  digests, confirming `19` tracks GA, not pre-release.
- Decision: default `POSTGRES_IMAGE=postgres:19.0`.
- Pulled digest:
  `sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
- Server reports: `PostgreSQL 19.0 (Debian 19.0-1.pgdg13+1)`.
- Image default `PGDATA=/var/lib/postgresql/19/docker` (verified via
  image inspect); the Compose file mounts the data parent and does not
  override the image default.

Override with `POSTGRES_IMAGE`; `dev:db:pull` refuses RC/beta/alpha
tags. Re-verify tags before changing the default: release schedules
and Docker tag promotion can differ.

## Start and stop

```sh
task --taskfile Taskfile.dev.yml dev:db:up     # secrets, checks, start, wait
task --taskfile Taskfile.dev.yml dev:db:status # container + server version
task --taskfile Taskfile.dev.yml dev:db:logs   # secret-safe logs
task --taskfile Taskfile.dev.yml dev:db:down   # stop, keep data
```

## Volumes and major versions

Data lives in `miniapp-pgdata-<MAJOR>` (currently
`miniapp-pgdata-19`). Startup refuses when the selected image major
does not match the existing volume, without deleting anything:

- `postgres-volume-check.sh` compares the image major against
  existing `miniapp-pgdata-*` volumes.
- Mismatch exits non-zero with migration instructions.
- `MINIAPP_ALLOW_FRESH_DB=1` explicitly permits a fresh empty volume;
  old volumes are preserved.
- `dev:db:reset` deletes only the current-major volume after a typed
  confirmation.

Major-version migration (dump/restore):

```sh
# 1. Dump from the old container while it still runs:
docker compose exec -T db pg_dumpall -U miniapp > pg19-backup.sql
# 2. Stop services, allow a fresh volume for the new major, start:
MINIAPP_ALLOW_FRESH_DB=1 task --taskfile Taskfile.dev.yml dev:db:up
# 3. Restore into the new instance, verify, then archive the old dump.
# 4. Remove the old volume only after verifying the restore.
```

## Host access

The base Compose file publishes no ports. For host tools and the
integration suite, the override binds loopback only:

```sh
docker compose -f compose.yaml -f compose.host.yaml up -d db
```

Default host port is 5433 (`MINIAPP_DB_HOST_PORT` overrides). Never
`0.0.0.0`.

## Integration tests

`task --taskfile Taskfile.dev.yml test:integration` starts the stack
with host access and runs `go test ./spikes/` against local
PostgreSQL. Database credentials come from the owner-only secrets
directory at runtime; values never appear in commands, logs, or test
output.
