# Frozen CLI Surface (Phase 0, per §13 keep-list)

One `miniapp` binary, two groups. Every lifecycle command maps to
exactly one §6.4 stage, owns no policy, calls shared services, and
never duplicates `flyctl`, `psql`, or platform admin tools (A14).

## Invoke group (thin REST client)

- `miniapp invoke <id> <tool> --json --idempotency-key ...`
- `miniapp status <run>`, `miniapp signal <run>`, `miniapp cancel <run>`

Stable `--help`, JSON output, and exit codes are contract.

## Lifecycle group

| Command | Stage | Calls |
|---|---|---|
| `init --class --profile` | Design | scaffolds manifest + profile from templates |
| `dev [--sprite]` | Develop | local server or isolated Sprite workspace |
| `validate` | Validate | manifest/schema/policy/placement/profile-digest admission |
| `test` | Validate | `testkit/` suites, unit through crash-boundary |
| `preview` | Validate | disposable Neon branch + Sprite preview |
| `publish` | Publish | immutable artifact + provenance + digests |
| `deploy` | Deploy | Machines services/workers, approved Sprite workspaces |
| `status`, `runs`, `logs` | Operate | one shared query path |
| `promote`, `rollback` | Deploy | governed release moves with recorded decisions |
| `doctor` | Operate | infrastructure, dependency, policy diagnostics |

New commands need ADR notes. This list is frozen at G0.
