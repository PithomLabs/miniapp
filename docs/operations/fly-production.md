# Fly Production Operations

Production is configured, never assumed. Until the owner confirms the
production application, every production task fails closed.

## Configuration

Non-secret environment configuration (names only, never values):

```sh
export MINIAPP_PROD_APP='<owner-confirmed production app>'
export MINIAPP_PROD_ORG='<expected production org>'
```

A Fly application named `miniapp-prod` exists but is explicitly
unwired: production commands must never silently default to it (or to
`miniapp-demo`). Wiring it requires explicit owner confirmation.

## Task reference

```sh
task --taskfile Taskfile.prod.yml doctor           # CLI, auth, app, org, revision
task --taskfile Taskfile.prod.yml fly:status       # inspect the app
task --taskfile Taskfile.prod.yml secrets:list     # metadata only
task --taskfile Taskfile.prod.yml secrets:stage SECRET=DATABASE_URL
task --taskfile Taskfile.prod.yml secrets:apply    # typed confirmation
task --taskfile Taskfile.prod.yml deploy           # checks + typed confirmation
task --taskfile Taskfile.prod.yml release:verify   # identity + health
task --taskfile Taskfile.prod.yml rollback:prepare # inspect only
```

## Fail-closed rules

Production tasks refuse to run when the application is unconfigured,
the organization mismatches, authentication is missing, required
checks fail, the artifact or revision is unidentified, or the
operator has not confirmed the exact target. Production secrets never
flow to local development, local tests, or development applications.

## Authorization

Deploying production requires an authenticated operator with Fly
release authority plus the typed in-task confirmation. The coding
agent holds no production deployment privileges: it may prepare and
inspect, never release.
