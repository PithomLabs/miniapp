One detail to handle carefully: as of November 22, 2026, PostgreSQL 19 is the newest stable PostgreSQL release, announced November 19. However, the Docker Official Image listing currently shows `latest` pointing to the 18.7 line and lists PostgreSQL 19 as release candidate tags.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

postgresql.org+2



So the implementation should verify the official image tags at execution time, use the newest stable GA image actually available, and never accidentally use a release candidate. It should also protect existing local database volumes when the image's major version changes.

Here is the implementation prompt. It keeps the design deliberately small: two independent Taskfiles, one local PostgreSQL Docker environment, Fly secrets for remote environments, and one consistent Go configuration convention.

# Implementation Task: Separate Development and Production Taskfiles, Docker PostgreSQL, and Fly Secrets

## Role

Act as the senior Go infrastructure engineer for `github.com/PithomLabs/miniapp`.

Implement a seamless, secure development and production environment workflow using:

- Go
- Task (Taskfile)
- Docker Compose
- Official PostgreSQL Docker image
- Fly.io and Fly CLI
- Neon PostgreSQL for remote development and production
- Existing `miniapp-demo` Fly application for development

Your objective is to make local development convenient while maintaining strict separation between development and production secrets, deployment targets, and operational commands.

Inspect the actual repository before making changes. Work with its existing architecture and configuration rather than introducing an unrelated infrastructure framework.

## 1. Governing requirements and boundaries

Use `docs/plan4.2.md` as the governing architecture baseline.

Preserve all approved architectural decisions and phase boundaries.

This task concerns developer experience, environment configuration, local PostgreSQL, Taskfile orchestration, and Fly secret management.

Do not implement Phase 1 runtime features or pull deferred Sprite, promotion, or production workflow functionality forward.

Do not modify `docs/plan4.2.md` to accommodate implementation convenience.

Do not invent a production Fly application name, organization, Neon project, credentials, or deployment configuration. Discover existing resources where possible. Require explicit owner configuration or action when something is missing.

Do not commit changes unless explicitly authorized.

## 2. Environment separation

Create exactly two independent Taskfiles:

- `Taskfile.dev.yml`
- `Taskfile.prod.yml`

Do not create a single Taskfile that imports or combines both environments. Shared implementation helpers may live under `scripts/`, but the environment-specific task definitions and permissions must remain separate.

The expected invocation convention is:

```
task --taskfile Taskfile.dev.yml --list
task --taskfile Taskfile.prod.yml --list
```

Verify the installed Task version and supported CLI syntax before implementing the convention.

### Development

Known Fly application:

`miniapp-demo`

Development consists of two separate execution targets:

1. Local development using Docker Compose and local PostgreSQL.
2. Remote development using the existing `miniapp-demo` Fly application and its development-only Fly secrets, connected to the development Neon project.

The default local development workflow must not require Neon credentials.

### Production

Production consists of:

1. A separately configured Fly application.
2. Its own Fly secrets.
3. A dedicated Neon production project and credentials.
4. Explicit production deployment and secret-rotation authorization.

Do not assume that the production application is named `miniapp-prod`. If it does not already exist, require the owner to configure the actual application name and organization. Never create production infrastructure automatically.

## 3. Local development PostgreSQL

Use Docker Compose to run the official PostgreSQL Docker image.

The local development database must be independent of Neon and must work without a remote database connection.

### Image selection

The requirement is to use the newest stable PostgreSQL release available through the official Docker image.

At implementation time:

1. Check the official PostgreSQL release information.
2. Check the actual official Docker image tags.
3. Select the newest stable GA image that is available.
4. Never use an RC, beta, or development image.
5. Record the exact resolved image tag, actual server version, and image digest in the developer documentation and `task dev:doctor` output.

Support overriding the image using a non-secret configuration variable such as `POSTGRES_IMAGE`.

Do not silently assume that Docker's `latest` tag always means the newest PostgreSQL GA release. The tag and release publication schedules can differ.

### Major-version and persistent-volume safety

A floating image tag introduces a significant local-development hazard: a major PostgreSQL update may make an existing data directory incompatible.

Solve this at the source.

- Detect the actual PostgreSQL major version of the selected image.
- Ensure database storage is isolated by major version, or refuse startup when the existing volume is incompatible.
- Never delete an existing database volume automatically.
- Never silently attempt an in-place major-version upgrade.
- Preserve old volumes until the developer explicitly migrates, archives, or deletes them.
- Document a safe logical dump/restore procedure for major-version changes.
- Pull the selected image deliberately through a Task command; do not make every ordinary task unexpectedly pull a new major version.

Follow the official image's documented `PGDATA` and volume layout for the selected tag. Do not hardcode an obsolete storage path.

### Docker Compose requirements

Provide a Compose configuration with:

- PostgreSQL service.
- Health check using `pg_isready`.
- Named persistent volume with major-version protection.
- Dedicated Compose network.
- No publicly exposed database port by default.
- Optional host access bound explicitly to `127.0.0.1`, never `0.0.0.0`.
- Restart behavior appropriate for local development.
- Predictable database, role, and schema initialization.
- No production credentials or Neon URLs.

Use the PostgreSQL image's supported file-based initialization credential mechanism where appropriate. Verify support against the selected image.

Do not put database passwords in Dockerfile `ARG`, Dockerfile `ENV`, Compose literals, command-line arguments, or tracked files.

## 4. Go configuration: environment variables and secret files

Provide one configuration contract that works both locally and on Fly.

For database credentials, support:

```
DATABASE_URL
DATABASE_URL_FILE

DBOS_SYSTEM_DATABASE_URL
DBOS_SYSTEM_DATABASE_URL_FILE
```

Rules:

1. When the ordinary variable is set, use its value.
2. When the `_FILE` alternative is set, read the referenced file.
3. If both are set for the same credential, return a configuration error. Do not silently choose one.
4. If neither exists, fail with a sanitized and actionable error.
5. Never include the secret value in errors, logs, diagnostics, or test failure output.
6. Validate that the resolved value is nonempty.
7. Do not log the resolved connection URL.
8. Do not expose credentials through CLI diagnostic commands.

Keep this implementation small and reusable. Do not build a generic secret-management framework.

Add unit tests for every precedence, missing-value, unreadable-file, conflicting-variable, whitespace, and redaction case.

Do not introduce production behavior changes without tests.

## 5. Local secrets and Docker Compose

Local-only secrets must not be stored in the repository.

Use a protected host directory outside the repository, following the platform's appropriate user-specific configuration or state directory.

Requirements:

- Directory permissions restricted to the current owner.
- Secret files readable only by the owner.
- Directory excluded from Git.
- No secret values in `.env`, `.env.example`, shell history, Taskfile variables, or process arguments.
- Generate local-only credentials using an appropriate cryptographically secure random generator.
- Preserve the credential across normal container restarts.
- Do not overwrite existing credentials during ordinary startup.
- Never reuse a local development credential in Fly development or production.

Compose file-backed secrets should deliver credentials to the intended containers.

For the PostgreSQL container, use the official image's supported `POSTGRES_PASSWORD_FILE` mechanism if applicable.

For an application container, use `DATABASE_URL_FILE` and `DBOS_SYSTEM_DATABASE_URL_FILE` where those credentials are needed.

Do not mount the host's entire secret directory into containers. Mount only the individual secret files each service requires.

Important security qualification:

Docker Compose file-backed secrets are not equivalent to an encrypted hardware-backed vault. They are local file mounts. A process with sufficient host-user privileges, Docker daemon access, or access to a secret-bearing container may still obtain the secret.

Do not claim protection against arbitrary malicious code running with those privileges. Document the actual threat boundary honestly. Restrict agent access to secret directories and privileged Docker operations wherever the local execution environment supports that separation.

## 6. Fly secrets management

Fly is the authoritative runtime secret store for remote development and production applications.

Development:

`miniapp-demo`

Production:

The explicitly configured production Fly application.

Use separate values for every environment.

Expected logical secret names:

```
DATABASE_URL
DBOS_SYSTEM_DATABASE_URL
```

Additional secrets should be added only when an actual component requires them.

### Secret-management rules

- Use `fly secrets list` only for metadata inspection.
- Never attempt to retrieve existing secret values.
- Never print, echo, trace, or persist secret values.
- Use supported Fly CLI standard-input mechanisms for secret import or staging.
- Never place secret values in shell command arguments.
- Disable shell tracing in credential-handling scripts.
- Use hidden terminal input for manual provisioning.
- Keep staged-secret and deployment operations separate where Fly supports it.
- Confirm deployment/restart implications before applying secret changes.
- Require explicit production approval before changing production secrets.
- Do not give the coding agent unrestricted access to Fly authentication or production deployment authority.

Inspect the installed Fly CLI version and its current `secrets` command help before implementing exact commands. Do not invent unsupported options.

### Fly secret provisioning scripts

Implement a small shared helper for interactive secret provisioning.

It must:

1. Accept an explicit environment.
2. Resolve the exact allowed Fly application from a non-secret environment configuration.
3. Display the target application and organization before accepting input.
4. Accept only an allowlisted secret name.
5. Read the secret without terminal echo.
6. Pass it through standard input using the verified Fly CLI mechanism.
7. Avoid writing the value to disk.
8. Disable shell tracing.
9. Clear local shell variables after use.
10. Report only the secret name, target application, and operation result.

Production provisioning must require an additional explicit confirmation that identifies the exact production application.

Do not create a general arbitrary-secret or arbitrary-command injection mechanism.

## 7. Taskfile development interface

`Taskfile.dev.yml` exclusively owns development operations.

Implement an appropriately small task surface:

| Task                | Responsibility                                                          |
| ------------------- | ----------------------------------------------------------------------- |
| `doctor`            | Check Docker, Compose, Go, Task, Fly CLI, and development configuration |
| `dev:db:pull`       | Pull the selected official PostgreSQL image                             |
| `dev:db:up`         | Start local PostgreSQL with safe volume handling                        |
| `dev:db:down`       | Stop local services without deleting database data                      |
| `dev:db:logs`       | Show database logs with secret-safe diagnostics                         |
| `dev:db:status`     | Show container and PostgreSQL health/version                            |
| `dev:db:reset`      | Explicitly destructive local reset with confirmation                    |
| `dev:up`            | Start the normal local development stack                                |
| `dev:down`          | Stop local development                                                  |
| `test`              | Run local unit and contract tests                                       |
| `test:integration`  | Run integration tests against local Docker PostgreSQL                   |
| `fly:status`        | Inspect `miniapp-demo`                                                  |
| `fly:secrets:list`  | Inspect development secret metadata                                     |
| `fly:secrets:stage` | Interactively stage a development secret                                |
| `fly:secrets:apply` | Apply staged development secrets                                        |
| `fly:deploy`        | Deploy to the existing development Fly app                              |

Do not add tasks that expose secret values.

Do not make local development depend on production or Neon credentials.

## 8. Taskfile production interface

`Taskfile.prod.yml` exclusively owns production operations.

It must not import `Taskfile.dev.yml`, start local PostgreSQL, or expose development tasks.

Implement a deliberately smaller and stricter task surface:

| Task               | Responsibility                                                                        |
| ------------------ | ------------------------------------------------------------------------------------- |
| `doctor`           | Verify Fly CLI, configured production app, organization, and production preconditions |
| `fly:status`       | Inspect the production application                                                    |
| `secrets:list`     | List production secret metadata                                                       |
| `secrets:stage`    | Interactively stage an approved production secret                                     |
| `secrets:apply`    | Apply production secrets after explicit confirmation                                  |
| `deploy`           | Deploy production after all required checks and owner confirmation                    |
| `release:verify`   | Verify deployed release identity and health                                           |
| `rollback:prepare` | Inspect and prepare rollback; never execute destructive action silently               |

Production tasks must fail closed if:

- The production application is not explicitly configured.
- The Fly organization does not match the expected production organization.
- The current authenticated identity lacks the required authority.
- The required build or validation checks fail.
- The deployment artifact or source revision cannot be identified.
- The operator has not explicitly confirmed the target.

Production deploy must never silently default to `miniapp-demo`.

Do not grant the coding agent production deployment privileges.

Do not make production secrets available to local development, local tests, or development Fly applications.

## 9. Shared implementation helpers

Share only environment-neutral mechanics.

Suggested structure, subject to actual repository conventions:

```
Taskfile.dev.yml
Taskfile.prod.yml

compose.yaml

scripts/
  dev/
    postgres-image.sh
    postgres-volume-check.sh
    local-secrets-init.sh
  fly/
    secret-input.sh
    app-preflight.sh
    secret-apply.sh
    release-verify.sh

docs/
  development/
    local-postgres.md
    fly-development.md
    secrets-management.md
  operations/
    fly-production.md
    secret-rotation.md
```

Do not create this exact tree mechanically if an equivalent structure already exists.

Shared scripts must accept explicit environment and application parameters, validate them against an allowlist, and fail closed.

The development and production Taskfiles remain independent even when they call shared helpers.

## 10. Security and correctness tests

Add regression tests and static checks for:

### Environment isolation

- Development tasks cannot target the production application.
- Production tasks cannot default to development.
- Production task definitions do not expose local destructive database operations.
- Local Compose never receives Neon production credentials.
- Environment configuration cannot silently cross-contaminate.

### Secret handling

- No secret is printed by secret-management scripts.
- No secret appears in process arguments.
- Shell tracing is disabled in sensitive scripts.
- Missing secrets and invalid permissions fail safely.
- Secret values are absent from captured normal logs and error messages.
- Docker Compose references secret files rather than literal credentials.
- Secret paths are excluded from Git.

### Database safety

- PostgreSQL starts and becomes healthy.
- `DATABASE_URL_FILE` works.
- `DBOS_SYSTEM_DATABASE_URL_FILE` works.
- Local migrations and tests run against local PostgreSQL.
- Existing local data survives normal stop/start.
- PostgreSQL major-version changes do not reuse an incompatible data directory.
- Reset requires explicit confirmation.
- No automatic volume deletion occurs.

### Fly safety

- Development secrets target only `miniapp-demo`.
- Production secret operations require the configured production target and confirmation.
- Secret listing never attempts to read values.
- Invalid or missing app configuration fails closed.
- Secret rotation does not silently trigger an unexpected production deployment.

### Taskfile safety

- Both Taskfiles parse and list successfully.
- All documented task names exist.
- The two task inventories remain distinct.
- Task commands do not interpolate secrets into shell command strings.
- Destructive operations are explicitly gated.

## 11. Required documentation

Provide a concise developer guide explaining:

1. Required tools and versions.
2. How to start local PostgreSQL.
3. How the official PostgreSQL image is selected and updated.
4. How major-version-specific data volumes are managed.
5. How Go reads `*_FILE` credentials.
6. How to initialize local-only secrets.
7. How to use `Taskfile.dev.yml`.
8. How development Fly secrets are managed.
9. How `Taskfile.prod.yml` differs.
10. How production authorization and confirmation work.
11. How to rotate secrets without exposing them.
12. What the local security boundary can and cannot guarantee.

Provide concrete commands, but never document or invent credential values.

## 12. Verification

Run and report the actual results of:

- Taskfile parsing and task listing for both environments.
- Shell syntax validation for all new scripts.
- `gofmt`.
- `go vet ./...`.
- `go test ./...`.
- Existing schema and contract drift checks.
- Docker Compose configuration validation.
- Local PostgreSQL image pull and startup, where Docker is available.
- Health check and database connectivity.
- Local migrations and integration tests.
- Secret-handling regression tests.
- Environment-separation tests.
- Git status and diff review.

If Docker is unavailable, classify Docker execution as BLOCKED and provide reproducible instructions. Do not claim a local PostgreSQL test passed merely because the Compose file parses.

Do not weaken existing tests.

Do not print environment variables or database connection strings when capturing verification output.

## 13. Deliverables

At completion, report:

1. Actual repository state before implementation.
2. Files created or modified.
3. `Taskfile.dev.yml` task inventory.
4. `Taskfile.prod.yml` task inventory.
5. Docker PostgreSQL image selection and resolved version.
6. Go configuration changes and regression tests.
7. Secret-delivery and authorization design.
8. Verification commands and actual results.
9. Security limitations and residual risks.
10. Any owner action needed for production Fly configuration.

Do not commit changes.

## Completion standard

The desired user experience is:

Development:

```
task --taskfile Taskfile.dev.yml doctor
task --taskfile Taskfile.dev.yml dev:db:up
task --taskfile Taskfile.dev.yml test
task --taskfile Taskfile.dev.yml test:integration
task --taskfile Taskfile.dev.yml dev:down
```

Remote development:

```
task --taskfile Taskfile.dev.yml fly:status
task --taskfile Taskfile.dev.yml fly:secrets:list
task --taskfile Taskfile.dev.yml fly:deploy
```

Production:

```
task --taskfile Taskfile.prod.yml doctor
task --taskfile Taskfile.prod.yml fly:status
task --taskfile Taskfile.prod.yml secrets:list
task --taskfile Taskfile.prod.yml deploy
```

Production operations must remain explicit and human-authorized.

**The governing principle: one Go application configuration contract, one local Docker PostgreSQL workflow, two independent Taskfiles, isolated environment credentials, and no secrets committed to or printed by the development workflow.**

The main design choice is that local development uses Docker PostgreSQL and does not need Neon credentials. Fly development and production each receive their own Neon connection credentials at runtime. This avoids copying cloud secrets onto the developer's machine just to run ordinary tests.