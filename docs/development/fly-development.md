# Fly Development (`miniapp-demo`)

Remote development uses the existing `miniapp-demo` Fly application
(org `personal`, verified present). Development Fly secrets connect
it to the development Neon project; values are managed per
`secrets-management.md` and never appear below.

## Inspect

```sh
task --taskfile Taskfile.dev.yml fly:status        # preflight + status
task --taskfile Taskfile.dev.yml fly:secrets:list  # metadata only
```

## Stage and apply secrets

Staging writes the value without deploying; applying restarts machines:

```sh
task --taskfile Taskfile.dev.yml fly:secrets:stage SECRET=DATABASE_URL
task --taskfile Taskfile.dev.yml fly:secrets:apply
```

Staging prompts for the value with terminal echo disabled and passes
it to `flyctl secrets import` over stdin. Applying shows secret
metadata, asks for confirmation, then runs `flyctl secrets deploy`.

## Deploy

```sh
task --taskfile Taskfile.dev.yml fly:deploy
```

The task runs preflight (app exists and is accessible), then
`flyctl deploy --app miniapp-demo`. It fails closed without a
`fly.toml`, which arrives with Phase 1 deployment work. No Machines
were created or released by the prompt4 implementation task.
