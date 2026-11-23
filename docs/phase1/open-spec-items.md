# Phase 1 Open Spec Items

Normative gaps found during Phase 1 implementation. Nothing here is
assumed; each item needs an owner decision or an approved-spec citation
before the affected behavior is implemented.

## OSI-1: lead-followup exact business semantics — OPEN

No approved source (plan4.2, prompt, contracts, fixtures) specifies the
follow-up sequence: number/timing of touches, channels, quiet hours,
exit and escalation conditions. Approved descriptors cover only
capabilities ("durable waits, eligibility re-checks, events,
cancel-safe recovery").

Impact: the `lead-followup` ADR draft must mark all sequence semantics
`[PROPOSED — requires owner confirmation]`; Class D implementation
waits for ADR human review per the hard-stop rule. Framework machinery
(DBOS executor, approvals, effects) proceeds independently.

## OSI-2: owner role vocabulary for approval scopes — OPEN

The framework enforces `ApproverScope` as a string match against the
decider's owner-defined roles (`auth.Binding.Roles`). No approved
source defines the role vocabulary (names, membership, who assigns
them). Tests use the synthetic value `tenant-ops`.

Impact: scope enforcement is mechanically proven; production role
content awaits the owner. No default roles invented.

## OSI-3: reviewed no-entity variance representation — OPEN

`plan4.2` §15.2 allows a consequential write to carry a reviewed
`no_entity` justification instead of an `entity_key`. Registry
admission currently requires `entity_key` for all effectful tools
because "reviewed" is a human judgment with no mechanical
representation yet.

Impact: effectful tools without an entity cannot register until
WS-J defines the variance representation (ADR reference + reviewer).
No such tool exists in Phase 1 scope.

## OSI-4: volumetric and payload-based limit semantics — OPEN

`BlastRadiusLimits` declares rate/volume caps, amount caps, and
recipient/domain allowlists, and plan4.2 §14.2 adds circuit
breakers and provider rate limits. No approved source defines:
counter windows and keys, amount formats/currencies, which
payload fields allowlists inspect, breaker thresholds and reset,
or provider-limit sources. WS-G enforces the generic,
unambiguous subset (live-grant revalidation, token binding,
kill switches at all five levels).

Impact: volumetric/allowlist/circuit enforcement awaits owner
decisions on formats and windows. The declarations validate
structurally; enforcement beyond WS-G scope is not claimed.

## Resolved notes (not open items)

- `docs/app_category_guidelines.md` is a concatenation of single-model
  drafts (`## claude`, `## gemini`, `## deepseek`, …), not a
  consolidated version. Per prompt10 §1 it is advisory only;
  `docs/plan4.2.md` governs on any conflict. No conflict affects
  Phase 1 scope (taxonomy A/B/C/D agrees).
- Sync results: per-tool `OutputSchema` is authoritative (owner
  decision D1=A). No shared sync envelope struct.
