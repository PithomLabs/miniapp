# G0-Core Adjudication (prompt8, 2026-11-22)

Acceptance review of Phase 0 against `docs/plan4.2.md` §23.1. Governing
baseline and review are unchanged (checksums verified before and after).
Owner inputs used as adjudication evidence only: `docs/prompt7_imp.md`
(accepts conditions 3/5 evidence) and `docs/roadmap.md` (position table);
neither invents Q7/Q8/Q9, harness, or commit decisions.

## 1. Condition matrix

| # | Governing requirement (plan4.2 §23.1) | Implementation | Behavioral evidence | Status | Gap | Owner dependency | Completion evidence | Next action |
|---|---|---|---|---|---|---|---|---|
| 1 | Core A/D contracts, identity/authority, sync/async invocation, effect record, entity-key, registry model, release binding reviewed and versioned | `contract/` (manifest, capability, `Caller`, `RunHandle`, effect machine, `entityKey`/`noEntityReason`, `ReleaseEnvelope.Matches`), 11 generated schemas + drift tests, ADR-002/004/005/009/016 | `go test ./...` green 21:04; no human review recorded | PARTIAL | Owner review sign-off; review must also settle whether per-tool `OutputSchema` suffices for sync results (no shared sync envelope struct exists) vs requiring one | Contract-review sign-off | Dated acceptance record naming the reviewed revision | Owner reviews and signs |
| 2 | Phase 1 deps pinned: version, license, date, upstream, owner | `docs/phase0/version-register.md`, `go.mod`/`go.sum` | Resolves, builds, vets clean; register has all five fields per row incl. upgrade owners | PASS | None | None | — | None |
| 3 | DBOS/Neon transaction semantics at the shared pool-handle boundary, crash probes | Spikes + pinned DBOS v1.9.0 | Neon 20:10–20:15 (commit/rollback/crash/family, separate pools) + prompt7 shared-handle test with detection proof + negative control, local and Neon 20:42–20:43, re-confirmed 21:04 | PASS | None (owner accepted per `prompt7_imp.md`) | None | — | None |
| 4 | Per-family AppName, version policy, retention/drain, proven-or-revised deployment topology | `docs/phase0/dbos-family-plan.md`, ADR-016, AppName isolation spike | AppName isolation PASS on Neon 20:10; version/retention documented; Fly overlap drill explicitly G6 | PARTIAL | Owner adjudication of the G0 design requirement (see §2 below) | Condition-4 adjudication | Recorded decision accepting the topology + deferral, or a revised unit | Owner adjudicates |
| 5 | Tenant tx-local context + RLS under connection mode and concurrency | `internal/store.WithTenant`, RLS + concurrent spikes | NOBYPASSRLS, default-deny, no-leak on Neon 20:15; concurrent barrier test PASS local (`-race`) and Neon 20:42–20:43; negative controls demonstrated | PASS | None (owner accepted per `prompt7_imp.md`) | None | — | None |
| 6 | Initial MCP/REST round trip with first target harness | MCP router design; in-process SDK spike | Protocol round-trip PASS fresh (local + Neon sessions); no harness named, no harness round trip | PARTIAL | Harness selection + minimal acceptance round trip | First-harness naming | Named harness + observed invocation result | Owner names harness; then one round-trip check |
| 7 | Named security owner signs threat model | `docs/phase0/threat-model.md` (draft, unsigned) | None (signature absent) | BLOCKED | Q9: name + signature with role, revision, date, reservations | Q9 | Signed record | Owner names and signs |
| 8 | prompt2 remainder or Appendix D acceptance | Appendix D (D1–D4, architect-authored) | None (no remainder supplied, no acceptance recorded) | BLOCKED | Q8: supply text or explicitly accept Appendix D as governing replacement | Q8 | Decision record | Owner decides |
| 9 | ADR-001 Go port + Class C/D decision records + human approval duties | `docs/adr/001-dbos-adoption.md`, `miniapps/adr001-template.md`, `inventory.md` | Design artifacts verified present and aligned | PASS | None | None | — | None |
| 10 | A15/A16 designs recorded | ADR-016/017, CLI spec, golden-path tests, §23.1(12–13) | Designs present in frozen baseline; operational path correctly deferred to G6 | PASS | None | None | — | None |
| 11 | Dated per-spike reports with revision, env, commands, results, dispositions | `docs/evidence/` entries; dispositions in frozen §23.2/§23.3 | Complete except source revision: zero commits, no commit authorized | PARTIAL | First-commit authorization, then bind each report to the revision | First commit | Commit hash referenced from every entry | Owner authorizes; then re-run evidence-producing checks at that revision |

## 2. Condition 4 four-part adjudication

**Behaviorally proven.** Two AppNames on one Neon branch complete the
same workflow shape independently with correct results
(`TestDBOSFamilyIsolation`, Neon 20:10); shared-handle commit/rollback
and crash recovery pass on Neon. AppName namespace isolation is
established; nothing in the evidence contradicts per-family execution.

**Documented.** `dbos-family-plan.md` defines: stable per-family
`AppName`; application version tied to the workflow compatibility line;
old artifacts retained until pending workflows drain or migrate under a
verified patch strategy; default topology of one worker process group
per family on the same Go image with a family selector; revision rule
falling back to separate Fly apps if process-group isolation cannot
preserve per-family versions.

**Must be decided now (owner).** Whether the documented version policy,
retention/drain strategy, and same-image process-group topology are
accepted as the G0 deployment strategy, with the residual risk (topology
fails under real Fly overlap conditions) carried explicitly to G6 and
the revision rule as its tripwire. Accepting means: the G0 design
requirement is satisfied; rejecting means: revise the deployment unit
now (e.g., separate Fly apps per family by default).

**Deferred to G6 (not waived).** Live Fly deployment overlap, graceful
shutdown and worker drain, actual deployment/recovery interaction, and
proof that incompatible versions cannot race. G6 owns these proofs;
this adjudication does not claim them.

## 3. Question classification (Appendix B vs §23.1)

Q7 (CLI assessment keep-list) appears in Appendix B with no G0 gate
assignment, and none of the eleven §23.1 conditions requires CLI
sign-off; condition 10 requires only that the A15/A16 designs be
recorded, which they are. Q7 is therefore **Phase 1 preparation, not
a G0 blocker**. Same discipline applied:

| Item | Governing source | Classification |
|---|---|---|
| Q1 first SaaS | Appendix B ("Gates Phase 2"), §23.2 G2 | Phase 2, not G0 |
| Q2 two-family matrix | §23.1(6) ("remains a Phase 6 gate") | Phase 6; first-harness naming alone is G0-6 |
| Q3 Enterprise budget | Conditional on wake-test evidence | Later; not G0 |
| Q4 Neon | Appendix B + §23.1(3)(5) | Resolved for G0 (probes executed) |
| Q4 Fly org/region | Appendix B ("before G6 deployment drills") | Later Fly work, not G0 |
| Q4 Sprite org/quota | Appendix B ("before Phase 4") | Phase 4, not G0 |
| Q5 plan3_review | No governing requirement found | Housekeeping, not G0 |
| Q6 agent-toolchain | Catalog scope, Phase 5 item | Not G0 |
| Q7 CLI sign-off | Appendix B, no §23.1 condition | Phase 1 preparation, not G0 |
| Q8 Appendix D | §23.1(8) | **G0 blocker** |
| Q9 security owner | §23.1(7) | **G0 blocker** |

## 4. Owner decision packet

### A. Mandatory G0 decisions

1. **Q8 — prompt2 remainder.** `prompt2.md` ends at §7.1; the remainder
   was never supplied; Appendix D holds architect-authored completions
   D1–D4 that are not original prompt text. Accepting Appendix D makes
   those completions the governing replacement. Alternatives: supply
   the missing text (preferred if it exists) or accept Appendix D.
   Evidence needed: dated decision record. Unblocks: condition 8.
2. **Q9 — security owner.** No owner is named; the threat model is an
   unsigned draft. Name the accountable individual (not inferred from
   authorship), who then signs the exact threat-model revision with
   role, date, and any reservations. Unblocks: condition 7.
3. **First harness.** Condition 6 needs one named harness plus a minimal
   observed invocation round trip (auth → invoke → run handle/result).
   Name it; the round trip is then a small bounded check, not the G6
   matrix. Unblocks: condition 6.
4. **Condition 1 review sign-off.** Contracts are implemented and green
   but unreviewed, including the open reading that per-tool
   `OutputSchema` (with no shared sync envelope struct) governs sync
   results. Review and sign with the reviewed revision identified, or
   require the envelope. Unblocks: condition 1.
5. **Condition 4 adjudication.** Accept the §2 topology + deferral as
   the G0 strategy, or revise the deployment unit now. Unblocks:
   condition 4.
6. **First commit.** Authorize the initial commit per §5 below so every
   dated report can bind to a source revision. Unblocks: condition 11.

### B. Classified non-blockers

Q7 (Phase 1 preparation), Q1 (Phase 2), Q2 matrix (Phase 6), Q3
(conditional), Q4 Fly/Sprite (later), Q5 (housekeeping), Q6 (Phase 5
catalog). None may hold G0-Core hostage.

## 5. First-commit request

- **Scope:** the entire working tree as inventoried (140 files, including this adjudication), minus
  owner-secret material (none lives in the repo; secrets stay in Fly
  and the owner-only state dir). No path needs exclusion beyond the
  existing `.gitignore`.
- **Benefit:** establishes the source revision every condition-11
  report must cite; converts "pending first commit" entries to
  verifiable evidence.
- **Docs inclusion:** yes — the evidence value lies in the
  code-plus-contracts-plus-evidence tree as one reviewed unit.
- **Pre-commit verification:** `gofmt -l` empty, `go vet ./...` clean,
  `go test ./...` green, `scripts/checks.sh` pass, both Taskfile
  inventories parse, final file-inventory diff reviewed, then commit
  and re-run the evidence-producing checks at the resulting revision.

## 6. Verdict (superseded by §7; preserved for history)

**G0-Core NOT PASSED.** Five conditions pass (2, 3, 5, 9, 10); four
are partial (1, 4, 6, 11); two are blocked (7, 8). Technical evidence
is complete; every remaining gap is an owner decision, a signature, or
the authorized first commit. Technical readiness is high; formal
acceptance awaits §4 above. Phase 1 is not authorized by the
governing G0 rule until the verdict flips.

## 7. Prompt9 decisions received (2026-11-22 21:23 +08:00)

Decision-maker: Gani Mendoza (reviewer identity supplied in-session;
structured answers given in the same session are attributed to the
same respondent). No other owner input received; Q9 and harness
selection remain pending as of this record.

| ID | Decision | Wording accepted | Reservations |
|---|---|---|---|
| D1 | A. Accept current design | Per-tool OutputSchema authoritative; no shared envelope required | None stated |
| D2 | A. Accept topology + G6 proof | Same-image process groups stand; Fly overlap/drain verified at G6; revision rule as tripwire | None stated |
| D4 | A. Accept Appendix D | D1–D4 govern in place of the missing prompt2 remainder | None stated |
| D6 | Approve 141-file commit scope | Scope, exclusions, checks, and message as presented; commit itself runs as a separate task | None stated |

Reviewed tree identified by file checksums (truncated SHA-256):

- contract/: capability.go 602cc9253afef4df, doc.go e9faecd5b61781bb,
  errors.go 70ff26551048208d, manifest.go 81d7e5936de7045f,
  manifest_test.go 82e20a97f0a152c0, policy.go 3b5981418a79aeb7,
  records_test.go 700020eff6290303, run.go 73dae83ae5e1fa38,
  schema.go a0acab00f8ec2bd3, schema_test.go cd245e212544a3ea,
  tasks.go 61e403a452edcc1e, validate.go c83c5bae3b81e205
- ADRs: 002 3a3e8ada1c6b06bc, 004 558dae0303ddc17b,
  005 e70c62a3480eabb6, 009 a0b768d5cc55366c,
  016 ee2c2c361d792407

## 8. Verdict (superseded by §9; preserved for history)

**G0-Core NOT PASSED.** Eight conditions pass (1, 2, 3, 4, 5, 8, 9,
10); two are partial (6, 11); one is blocked (7). Remaining: Q9
security owner (condition 7), first-harness naming plus round trip
(condition 6), and the authorized first commit with revision binding
(condition 11). Phase 1 is not authorized until the verdict flips.

## 9. Follow-up owner decisions (2026-11-22)

- **Q9 designation received.** Gani Mendoza, Architect, explicitly
  accepted security-owner responsibility. Threat-model signature not
  yet given; the approval question (approve
  `docs/phase0/threat-model.md` as written, without reservations?)
  is posed and awaiting an explicit yes/no.
- **First harness selected: OpenCode.** Round-trip execution
  (auth/invocation with observed result) is a separately authorized
  technical task, not yet run. Note: no OpenCode client is installed
  in the verification environment; the task plan must establish how
  the round trip is driven and observed.

## 10. Verdict (superseded by §11; preserved for history)

**G0-Core NOT PASSED.** Eight conditions pass (1, 2, 3, 4, 5, 8, 9,
10); three are partial (6, 7, 11); none blocked. Remaining: threat-model
signature (condition 7), OpenCode round-trip execution (condition 6),
and the authorized first commit with revision binding (condition 11).
Phase 1 is not authorized until the verdict flips.

## 11. Threat-model approval (2026-11-22 21:29 +08:00)

Gani Mendoza, Architect, approved `docs/phase0/threat-model.md` as
currently written, without reservations, in reply to an explicitly
worded approval request. An earlier ambiguous single-letter reply was
refused as insufficient and clarified before recording; nothing was
inferred.

## 12. Verdict (superseded by §13; preserved for history)

**G0-Core NOT PASSED.** Nine conditions pass (1, 2, 3, 4, 5, 7, 8, 9,
10); two are partial (6, 11); none blocked. Remaining: OpenCode
round-trip execution (condition 6) and the authorized first commit
with revision binding (condition 11). Phase 1 is not authorized until
the verdict flips.

## 13. Threat-model scope amendment (2026-11-22 21:32 +08:00)

Owner-directed one-line correction, applied verbatim: the scope line
referenced the superseded `docs/plan4.1.md`; it now references the
governing `docs/plan4.2.md`. Nothing else in the file changed
(checksums `8f708ac6e413f5dc` → `e4eed875c79fa51d`, truncated
SHA-256). The 21:29 approval covered the prior revision; re-approval
of the amended text is outstanding and explicitly requested, not
assumed.

## 14. Verdict (superseded by §15; preserved for history)

**G0-Core NOT PASSED.** Eight conditions pass (1, 2, 3, 4, 5, 8, 9,
10); three are partial (6, 7, 11); none blocked. Remaining:
threat-model re-confirmation (condition 7), OpenCode round-trip
execution (condition 6), and the authorized first commit with
revision binding (condition 11). Phase 1 is not authorized until the
verdict flips.

## 15. Threat-model re-confirmation (2026-11-22 21:33 +08:00)

Gani Mendoza approved the amended threat model (plan4.2 scope line,
checksum `e4eed875c79fa51d`) as written, without reservations, in
reply to an explicitly worded re-confirmation request.

## 16. Verdict (current)

**G0-Core NOT PASSED.** Nine conditions pass (1, 2, 3, 4, 5, 7, 8, 9,
10); two are partial (6, 11); none blocked. Remaining: OpenCode
round-trip execution (condition 6) and the authorized first commit
with revision binding (condition 11). Phase 1 is not authorized until
the verdict flips.
