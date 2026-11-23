# WS-E operator wiring — STOPPED: owner decision required

Date: 2026-11-23. Work order prompt28 §7 stops here by its own
terms: no suitable trusted operator authority exists, so the
Settle control path must not be wired by weakening the
boundary. The Settle crash tests (§7, first half) are complete;
only authenticated wiring is blocked.

## Source audit (why no existing authority qualifies)

1. Role vocabulary is explicitly OPEN (OSI-2):
   `ApproverScope` enforcement is mechanical string matching
   against `auth.Binding.Roles`, but "no approved source
   defines the role vocabulary (names, membership, who
   assigns them). Tests use the synthetic value `tenant-ops`."
   Using `tenant-ops` as the operator gate would invent the
   default role OSI-2 forbids — rejected by guardrail 2.
2. The API-key mechanism (credential → Binding) is trusted,
   but the keys file is demo scaffolding ("Production
   authentication policy arrives with a later phase" —
   `cmd/miniapp/main.go` header). Tenant ownership alone
   (the Cancel pattern) has no deny leg for operator scope
   and fails prompt28's explicit "administrative authority"
   requirement.
3. Authority kinds (`user`/`service`/`trigger`) exist as
   vocabulary (`contract/policy.go`), but no approved rule
   makes any kind an operator gate; key files can mint any
   kind. Treating one as administrative would invent the rule.
4. The D1 control-plane pattern (owner-only DB function +
   separate `switch` command + control DSN) is spec-only:
   no `switch` verb, no `MINIAPP_CONTROL_DATABASE_URL`
   implementation, no X-15 tests exist. The settle doc-spec
   itself says no control DSN is needed (effect tables are
   app-writable), so D1 does not transfer to Settle anyway.

## Smallest architecture decision required

The owner must designate ONE operator authority (in
increasing order of machinery):

- (a) Close OSI-2 for this purpose: approve an operator
  role name, its membership/assignment, and require it
  (server-side `Roles` check) on the settle path.
- (b) Designate an authority-kind rule (e.g. a dedicated
  operator kind distinct from user/service/trigger) with
  issuance rules for the keys file.
- (c) Direct that tenant-authenticated settle with
  audit-only attribution is acceptable (weakest; explicitly
  approves the reduced boundary).

Until then: no `settle` verb, no service-level settle
method, no operator-identity plumbing. `effects.Settle`
remains test-only Go API (correctly unwired — verified in
the WS-F inventory).

## Collateral note (non-blocking)

The F-03 doc-spec bullet (contract §12) still describes the
`UNKNOWN→RECONCILING→terminal` pair; RECONCILING was removed
(P2-open §3.3) and §7/M3 already reflects atomic Settle.
When wiring lands, that bullet needs a one-line correction;
it is left untouched here as a historical record.

## Addendum — decision received and implemented (prompt28.1)

The owner approved Option A: canonical role
`miniapp.settle.operator`, exact case-sensitive match
against `auth.Binding.Roles`, tenant from the binding,
server-derived operator identity, deny by default. The
stop above is lifted on exactly those terms; nothing in
the original audit is rewritten.

Implemented: `runtime.Service.SettleEffect` (single shared
boundary; no pre-read — tenant isolation rides the
predicated operation; missing and foreign effects map to
one terminal `unknown-effect`), `miniapp settle` CLI verb
(exact F-03 flags; no independent authorization logic),
F-03 one-line atomicity correction, contract §16 row,
evidence matrix (`TestP281SettleOp*`,
`TestP281SettleVerb*`). `effects.Settle` untouched.

Carried forward (not resolved here): demo API-key
provisioning is not production identity (prompt28.1 §4);
CLI `--credential`/env credential is visible in process
listings — documented, proper resolution belongs to the
future Miniapp Security Contract alongside role issuance,
membership changes, revocation, and identity propagation.
