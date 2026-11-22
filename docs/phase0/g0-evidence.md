# G0 Evidence Tracker (plan4.2: G0-Core + phase gates)

G0-Core passes only when every §23.1 condition below is green with dated
behavioral evidence. Extension proofs (River, SaaS, Sprite, Fly) belong
to their phase gates and never hold G0-Core hostage — but their
invariants are never waived either. No owner decision below is invented.

## Owner decisions

| ID | Decision | Gates | Status |
|---|---|---|---|
| Q9 | Name the security owner (signs threat model) | G0-Core-7 | RESOLVED 2026-11-22 21:29 +08:00 — Gani Mendoza, Architect; responsibility accepted and `docs/phase0/threat-model.md` approved as written, no reservations |
| Q8 | Supply prompt2 remainder or accept Appendix D | G0-Core-8 | RESOLVED 2026-11-22 21:23 +08:00 — Gani Mendoza accepted Appendix D (D1–D4) as the governing replacement; no exclusions stated |
| Q4-Neon | Neon project/region for G0-Core probes | G0-Core-3, G0-Core-5 | RESOLVED 2026-11-22 20:07 +08:00 — owner confirmed project `Pithom Labs` (`frosty-hat-76495512`, `aws-us-east-2`); disposable branch `phase0/g0-spikes` (`br-frosty-pine-bdwmx50b`) created with explicit approval; probes executed. Note: region differs from the earlier Fly `sin` note — co-location stays a G6 consideration |
| Q4-Fly | Fly org/region identified | Later (design in P0) | Identified — org `personal`, region `sin`, zero apps; deploy proof is G6 |
| Q4-Sprite | Sprite org/quota | Phase 4 | Pending — not a G0-Core requirement |
| Q1 | First SaaS integration | Phase 2 | Pending — visible owner action, not G0-Core |
| Q2 | Two harness families (name first harness early) | G6 (first needed for G0-Core-6) | First harness SELECTED 2026-11-22 — OpenCode; round-trip execution pending. Two-family matrix remains Phase 6 |
| Q3 | DBOS Enterprise budget stance | Wake-test fallback | Pending |
| Q5 | `plan3_review.md` fill or remove | Phase 0 cleanup | Pending |
| Q6 | `agent-toolchain` deferral | Catalog scope | Pending |
| Q7 | CLI keep-list approval | G0 CLI freeze | Pending |

## G0-Core conditions (§23.1) status

| # | Condition | Status | Evidence |
|---|---|---|---|
| 1 | Core A/D contracts reviewed/versioned | PASS | Reviewed and accepted by Gani Mendoza 2026-11-22 21:23 +08:00 (decision D1=A: per-tool OutputSchema authoritative, no shared envelope required); tree identified by file checksums (see adjudication); suite green; no reservations stated |
| 2 | Phase 1 deps pinned + verified | PASS | `docs/phase0/version-register.md` (pins, licenses, dates, owners); `go.mod` byte-identical to baseline, resolves, builds, vets clean |
| 3 | DBOS/Neon atomicity crash probe | PASS | Prior Neon evidence (commit/rollback/crash/family, separate pools, 20:10–20:15) plus prompt7 shared-handle evidence: `TestDBOSSharedHandle` + `TestDBOSSharedDetectionNegative` PASS local and Neon 20:42–20:43 +08:00 (detection log, commit/rollback durable state, step metadata; re-confirmed local 21:04). Owner accepted per `prompt7_imp.md`. See `docs/evidence/` |
| 4 | DBOS family AppName/version/drain plan | PASS | Topology accepted by Gani Mendoza 2026-11-22 21:23 +08:00 (decision D2=A: same-image process groups stand, Fly overlap/drain verified at G6, revision rule as tripwire); AppName isolation proven on Neon; version/retention plan documented |
| 5 | Tenant context/RLS probe | PASS | Prior Neon evidence (NOBYPASSRLS, tx-local context, default-deny, 20:15) plus prompt7 `TestConcurrentTenantIsolation` PASS local with `-race` and Neon 20:42–20:43 +08:00 (8 workers × 4 tenants, barrier overlap, durable per-tenant counts, unarmed negative leg; re-confirmed local 21:04). Owner accepted per `prompt7_imp.md`. See `docs/evidence/` |
| 6 | Initial MCP/REST round trip, first harness | PARTIAL | Harness selected: OpenCode (2026-11-22). In-process protocol evidence fresh; OpenCode auth/invocation round trip pending execution as a separately authorized technical task |
| 7 | Security owner signs threat model | PASS | Gani Mendoza approved the amended (plan4.2-scoped, `e4eed875`) revision as written 2026-11-22 21:33 +08:00; no reservations. Prior 21:29 approval covered the superseded revision; history preserved in adjudication |
| 8 | prompt2 remainder or Appendix D acceptance | PASS | Appendix D (D1–D4) accepted as governing by Gani Mendoza 2026-11-22 21:23 +08:00 (decision D4=A); no exclusions stated |
| 9 | ADR-001 port + decision template + inventory | PASS | `docs/adr/001-dbos-adoption.md` (Go port), `docs/adr/miniapps/adr001-template.md`, `inventory.md` |
| 10 | A15/A16 decisions recorded | PASS | ADR-016/017 + §18 golden-path tests + §23.1(12–13) in frozen baseline |
| 11 | Dated reports + failure dispositions per core spike | PARTIAL | Dispositions in frozen §23.2/§23.3; dated Neon reports recorded 20:10–20:16 +08:00 (see `docs/evidence/`); commit scope approved by Gani Mendoza 2026-11-22 21:23 +08:00 (decision D6) — source revision pending the separately authorized commit task |

## Deferred-gate prep (not G0-Core, tracked honestly)

| Proof | Gate | Status |
|---|---|---|
| River `InsertTx` atomicity + unique dedup | G3 entry | Proven on Neon direct 20:15 +08:00 (11.3s); Class C comparison + human review outstanding (Phase 3 artifact, not G0-Core) |
| Sprite provision/policy/exec/teardown | G4 entry | Harness written/compiling; needs Sprite token/org (Q4-Sprite); runbook below |
| Live SaaS + SSRF suite | G2 entry | Test design written (`ssrf-test-design.md`); needs Q1 + sandbox |
| Fly deploy overlap/drain | G6 | Design in family plan; needs Fly app + Neon restore drill |

## Incident note: Neon access lost mid-session (2026-11-22 ~16:20 +08:00)

Timeline (all +08:00): 16:05 projects listed (`openlevel-prod`,
`openlevel-staging`); branch `phase0/g0-spikes` created on staging;
connection strings issued; 16:19 direct psql OK (PG 18.7) and DBOS
atomicity tests PASSED with observed logs. Minutes later: direct PG
auth fails with 28P01 using the unchanged stored string, and the Neon
API reports zero projects under the same OAuth identity (token valid
to 17:12). No project/org mutation was issued from this session after
the branch create. Root cause is account-side and undetermined from
here (access revocation, project state change, or API-side fault).
Owner action: verify projects/access in the Neon console, restore
access, then re-run the spike suite to close G0-Core-3/5. The
16:19 DBOS passes stand as observed-once evidence only.

Addendum 2026-11-22 20:02 +08:00 (closure sprint, read-only probe):
`neon projects list` fails because the stored OAuth token has expired
and refresh requires writing the credential store, which is outside
this task's authorization and blocked in this environment. No refresh
was attempted, no credentials were copied or printed, and no remote
state was changed. Access remains unrestored; the owner action above
still stands.

Resolution 2026-11-22 20:04 +08:00: API access restored owner-side.
The visible project set changed: `openlevel-*` projects are no longer
listed; the account now shows `Pithom Labs` (`frosty-hat-76495512`,
`aws-us-east-2`) under the same identity. No conclusion is drawn here
about the earlier zero-project/28P01 state (account switch, org
change, or API-side fault are all consistent with observations); the
16:20 record above stands as written. Owner confirmed the new project
as the G0-Core target; all subsequent probes ran against a fresh
disposable branch on it.

## Spike runbooks (blocked spikes)

### Neon-backed spikes (DBOS, River, RLS)

Requires: restored Neon access (Q4-Neon); branch
`phase0/g0-spikes` (`br-frosty-pine-bdwmx50b`) on project `Pithom Labs`
(`frosty-hat-76495512`, owner-confirmed 2026-11-22 20:07 +08:00).

```bash
S=<writable secrets dir>
neon connection-string --project-id frosty-hat-76495512 \
  --branch phase0/g0-spikes > $S/neon_direct
cd /home/chaschel/Documents/go/miniapp
export TEST_DATABASE_URL="$(cat $S/neon_direct)" \
  TEST_DBOS_DATABASE_URL="$(cat $S/neon_direct)" \
  GOPATH=<writable> GOCACHE=<writable> GOTOOLCHAIN=local GOWORK=off
go test ./spikes/ -run 'TestDBOS|TestRiver|TestRLS' -v -timeout 12m
```

Expected: all PASS; DBOS crash test kills and relaunches a worker
mid-step. Record output, revision (`git rev-parse HEAD` after the
approved first commit), environment, and date in `docs/evidence/`.

### Sprite round-trip (G4 entry)

Requires: Sprite org with quota + `SPRITE_TOKEN` (Q4-Sprite).

```bash
export SPRITE_TOKEN='<org token>'  # plus Go env overrides above
go test ./spikes/ -run TestSpriteRoundTrip -v -timeout 8m
```

Expected: provision → deny-all policy apply → read-back match →
exec `echo` → teardown, all PASS. Never substitutes for A1–A3
framework controls (Phase 5).

## Sandbox build note

This machine needs `GOPATH`/`GOCACHE` overrides plus `GOWORK=off`
(read-only default Go paths; parent `go.work` exists) and direct
network for Neon (sandbox proxy carries only HTTP/S). CI runners on
clean checkouts need no overrides.
