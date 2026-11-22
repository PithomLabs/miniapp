# Secrets Management

## Local secrets

Local-only credentials live outside the repository in
`${XDG_STATE_HOME:-$HOME/.local/state}/miniapp` (override with
`MINIAPP_SECRETS_DIR`). Initialize once:

```sh
bash scripts/dev/local-secrets-init.sh
```

The directory is mode 700 and files are mode 600. Initialization never
overwrites existing credentials. Nothing under this directory is
committed: `.gitignore` excludes `.env`, key material, and secrets
directories as defense in depth.

Compose delivers `db_password` to PostgreSQL via
`POSTGRES_PASSWORD_FILE`. Only individually required secret files are
mounted; the directory itself is never mounted. Local credentials are
never reused on Fly.

## Go `_FILE` convention

Database credentials resolve through `internal/config`:

- `DATABASE_URL` or `DATABASE_URL_FILE`
- `DBOS_SYSTEM_DATABASE_URL` or `DBOS_SYSTEM_DATABASE_URL_FILE`

Direct value wins if only it is set; file content wins if only the
file is set; both set is an error; neither is an actionable error
naming the variables. Surrounding whitespace is trimmed. Errors,
logs, and diagnostics never contain values.

## Fly secrets

Fly is the runtime secret store for remote applications. Rules:

- `flyctl secrets list` shows metadata only; values are irretrievable.
- Values enter through `flyctl secrets import` over stdin, typed with
  terminal echo disabled; never as command arguments.
- Staging (`--stage`) and applying (`flyctl secrets deploy`) are
  separate explicit steps.
- Production additionally requires the application to be configured,
  the organization to match, and a typed application-name
  confirmation.
- Rotation: stage the new value, apply, verify health, then retire
  the old credential at the provider. See
  `../operations/secret-rotation.md`.

## Honest threat boundary

Docker Compose file secrets are local file mounts, not an encrypted
hardware-backed vault. Any process with the host user's privileges,
Docker daemon access, or access to a secret-bearing container can read
them. This design protects against accidental exposure (logs, shell
history, Git, process arguments, container images) — not against
malicious code running with those privileges. Restrict agent access
to secret directories and privileged Docker operations accordingly.
