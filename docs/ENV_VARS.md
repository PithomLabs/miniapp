# Environment Variables Reference

Every environment variable the miniapp project reads, in one place.
Names are identical across environments; only the source differs (local
files, process environment, Fly secrets). Nothing here carries a value —
see `docs/development/secrets-management.md` for handling rules.

Conventions: `REQUIRED` fails closed when unset; `OPTIONAL` has a safe
default or degrades gracefully. `SECRET: yes` means the value must never
appear in logs, errors, diagnostics, command arguments, or Git.

## Go runtime credentials (`internal/config`)

| Name | Default | SECRET | Notes |
|---|---|---|---|
| `DATABASE_URL` | — (or `_FILE`) | yes | Direct credential value. Mutually exclusive with `DATABASE_URL_FILE`. |
| `DATABASE_URL_FILE` | — (or direct) | no (path) | File holding the value. Both set, or neither, is a config error. |
| `DBOS_SYSTEM_DATABASE_URL` | — (or `_FILE`) | yes | Same pair semantics for the DBOS system pool. |
| `DBOS_SYSTEM_DATABASE_URL_FILE` | — (or direct) | no (path) | Same pair semantics for the DBOS system pool. |

Surrounding whitespace is trimmed; empty-after-trim fails. Errors name
variables and paths only. On Fly, `DATABASE_URL` / `DBOS_SYSTEM_DATABASE_URL`
arrive as injected secrets and the `_FILE` variants stay unset.

## Go runtime policy (`internal/config`)

| Name | Default | SECRET | Notes |
|---|---|---|---|
| `MINIAPP_POLICY` | unset → `warn`, or `strict` when `CI=true` | no | `strict`, `warn`, or `disabled` keep their meaning everywhere. Any other explicitly set value (unknown, empty, malformed, wrong case, surrounding whitespace) fails closed to `strict` regardless of `CI`. Only a genuinely unset variable takes the CI-dependent default. |
| `CI` | unset | no | Standard CI marker; `true` selects strict policy by default. Also set in CI. |

## Fly configuration (Taskfiles and `scripts/fly/`)

| Name | Default | SECRET | Notes |
|---|---|---|---|
| `MINIAPP_PROD_APP` | none (REQUIRED) | no (name) | Explicit production Fly application. Never defaults; unset fails closed. |
| `MINIAPP_PROD_ORG` | none (REQUIRED where checked) | no (name) | Expected production org; preflight refuses on mismatch. |

The dev target `miniapp-demo` is fixed in `Taskfile.dev.yml`, not
configurable. Fly secret *names* managed through the workflows are
`DATABASE_URL` and `DBOS_SYSTEM_DATABASE_URL` (allowlisted; values live
in Fly, never in this repo or its environment).

## Local development (Taskfiles, Compose, `scripts/dev/`)

| Name | Default | SECRET | Notes |
|---|---|---|---|
| `POSTGRES_IMAGE` | `postgres:19.0` | no | Official image tag. RC/beta/alpha tags are refused. |
| `POSTGRES_MAJOR` | `19` | no | Must match the image major; startup refuses on mismatch. |
| `POSTGRES_USER` | `miniapp` | no (name) | Bootstrap role created by the image entrypoint. |
| `POSTGRES_DB` | `miniapp_dev` | no (name) | Database created by the image entrypoint. |
| `MINIAPP_SECRETS_DIR` | `${XDG_STATE_HOME:-$HOME/.local/state}/miniapp` | no (path) | Owner-only secret directory (mode 700). Contents: `db_password` (mode 600). |
| `MINIAPP_DB_HOST_PORT` | `5433` | no | Loopback-only host port via `compose.host.yaml`. Base Compose publishes nothing. |
| `MINIAPP_ALLOW_FRESH_DB` | unset (`0` behavior) | no | Set to `1` to explicitly acknowledge starting a fresh volume beside preserved ones. |
| `XDG_STATE_HOME` | `$HOME/.local/state` | no | Platform standard; used only to derive the default secrets dir. |
| `HOME` | — | no | Platform standard; fallback for the default secrets dir. |
| `DBPW` | none (transient) | yes | Exists only inside `test:integration` to assemble test URLs; unset immediately after. Never exported by developers. |

## Spike and test configuration (`spikes/`, integration only)

| Name | Default | SECRET | Notes |
|---|---|---|---|
| `TEST_DATABASE_URL` | unset (test skips) | yes | River and RLS spikes. Set by `test:integration` from local secrets. |
| `TEST_DBOS_DATABASE_URL` | unset (test skips) | yes | DBOS atomicity, isolation, and crash spikes. |
| `SPRITE_TOKEN` | unset (test skips) | yes | Sprite round-trip spike. Phase 4 gate; never in CI. |
| `MINIAPP_SPIKE_WORKER` | unset | no | Internal: marks the crash-probe child process. |
| `MINIAPP_SPIKE_DIR` | unset | no (path) | Internal: crash-probe coordination directory (temp). |
| `MINIAPP_SPIKE_WFID` | unset | no | Internal: crash-probe workflow ID. |
| `MINIAPP_SPIKE_MARKER` | unset | no | Internal: crash-probe marker prefix. |

Skipped-for-missing-env is an explicit SKIP with reason, never a pass.

## Not environment variables

- `SECRET=name` is a Task CLI variable (`task ... SECRET=DATABASE_URL`),
  allowlisted to the two Fly secret names. It names a secret; the value
  is always read hidden from the terminal.
- `db_password` is a file name inside `MINIAPP_SECRETS_DIR`, not a variable.
