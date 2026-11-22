# Filled Version Register (Phase 0, verified 2026-11-22)

All pins verified against proxy.golang.org version lists plus
module-source inspection (go directives, licenses, and key APIs).
No `latest` anywhere. Re-verify on any upgrade.

| Role | Pinned source | License | Go floor | Upgrade owner | Evidence |
|---|---|---|---|---|---|
| Language toolchain | Go 1.26 (`go` directive; local toolchain go1.27.3) | — | 1.26 | Security owner (Q9) | Max floor across pins; `go.mod` |
| Durable workflows | `github.com/dbos-inc/dbos-transact-golang` v1.9.0 | MIT | 1.26 | Security owner (Q9) | `RunAsTransaction`, `RunAsStep`, `RunWorkflow`, `Sleep`, `Send`/`Recv`/`SetEvent`/`GetEvent`, queue API, `entitlements.go` + `WithSleepSuspend`/Conductor markers, LISTEN/NOTIFY in `dbos/internal/sysdb` (K5 mechanism confirmed) |
| Background jobs | `github.com/riverqueue/river` v0.51.2 (+ `riverdriver/riverpgxv5` v0.51.2) | MPL-2.0 | 1.26 | Security owner (Q9) | `InsertTx`/`InsertManyTx`/`Insert`, `UniqueOpts`/`JobArgs`/`InsertOpts`, `Client.Start/Stop`, `rivermigrate` up/down |
| Agent protocol | `github.com/modelcontextprotocol/go-sdk` v1.8.0 | Apache-2.0 | 1.25 | Security owner (Q9) | `mcp` client+server, stdio/SSE/Streamable transports, elicitation, `auth`/`oauth` packages; in-process round-trip passes (`spikes/mcp_test.go`) |
| Sprite operations | `github.com/superfly/sprites-go` v0.2.2 | MIT | 1.26 | Security owner (Q9) | Proxy zip matches inspected checkout (nested example/test-cli modules excluded; published go.mod pins client-signals v0.4.4 vs checkout v0.5.0 — no API impact); Appendix C re-verified at pin |
| Postgres driver | `github.com/jackc/pgx/v5` v5.13.0 | MIT | 1.25 | Security owner (Q9) | Satisfies DBOS (5.13.0) and River (5.11.0) floors |
| Schema generation | `github.com/invopop/jsonschema` v0.14.0 | MIT-style (COPYING) | 1.24 | Security owner (Q9) | `Reflector.Reflect` drives `contract.GenerateSchemas` + drift test |
| Migrations (tool) | `github.com/golang-migrate/migrate/v4` v4.20.1 | MIT | 1.25 | Security owner (Q9) | Pinned as CI tool (`go run .../cmd/migrate@...`); not imported |
| Key management | TBD — selected in Phase 2 with the vault decision | Verify then | — | Security owner (Q9) | A10/K10; envelope + KMS recommended, not yet chosen |

Upgrade ownership defaults to the security owner until Phase 1 names
module owners. Enterprise DBOS modules (Conductor, auto-upgrade) are
forbidden imports without an explicit owner-approved decision; the
import-boundary check enforces this.

## License notes

- River's MPL-2.0 is file-level copyleft: unmodified use as a Go
  dependency is fine. Do not modify River sources in the module
  cache or vendor tree; any patch must go upstream or trigger a
  license review.
- MCP SDK files carry the project's MIT→Apache-2.0 transition
  notice; treat the pin as Apache-2.0.
- Sprite gap APIs (connectors/usage/logs/privilege) remain a
  framework-owned REST client over the live API, verified when Q4
  provides access; no additional module pin needed.
