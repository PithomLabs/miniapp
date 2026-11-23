# Effect authoring: operation IDs and contention recovery

For miniapp authors whose handlers call `Perform`. Normative
rule: lifecycle contract §7/M3 ("C6 operation-ID rule").

## Derive BusinessOpID from stable business identity

Format: `tenant-id ":" business-key`, where `business-key`
identifies the business operation independently of any
particular execution:

- DO derive from stable business keys (entity ID, order
  number, subject + action).
- DO NOT derive from invocation IDs, run IDs, attempt IDs,
  timestamps, or random values.
- The SAME business operation retried under the SAME or a
  NEW idempotency key MUST reuse the identical
  BusinessOpID. Adoption, reconciliation, and
  exactly-once-per-operation dedup all key on it.

Attempt IDs are the framework's: every attempt epoch is
minted at claim/takeover time. Miniapps never supply them
(`Intent.AttemptID` is ignored).

## Handle ErrAttemptInProgress by returning promptly

When another attempt owns the operation, `Perform` returns
`ErrAttemptInProgress`. Inside a Class A handler:

1. Return the error immediately. Do not sleep, poll, or
   retry inside the handler — waiting while holding the
   Class A transaction is prohibited.
2. The framework rolls the whole attempt back (business
   writes included — nothing partially commits) and
   reports terminal contention. The invocation is NOT
   marked failed or successful.
3. The caller retries outside the transaction:
   - Pending/waiting records converge via same-key retry
     (the retry adopts the winner or reconciles safely).
   - Running Class A records answer `interrupted`: resubmit
     with a NEW idempotency key. The new attempt reuses
     the same BusinessOpID and adopts the winner's settled
     outcome without resending.
4. Recovery is bounded by the attempt lease (two minutes):
   a live attempt resolves or its row ages into
   takeover/quarantine. Never bypass or extend the lease
   from miniapp code.

Worked end-to-end proof: `TestP28C6ClassARecovery`
(runtime package) — contention inside a real handler,
rollback completeness, guidance codes, stable op IDs,
adopt-without-resend, and no held locks.

## Abandoned-PENDING boundary: a new BusinessOpID never adopts another invocation's pending effect

Precise failure scenario (P29-005):

1. A Class A invocation creates an effect in PENDING.
2. The process terminates during the claim-to-authorization
   window.
3. The owning invocation is abandoned.
4. A different invocation retries using a new BusinessOpID.
5. Cross-invocation adoption is refused because the pending
   effect remains bound to its original invocation.
6. The operation can remain blocked until same-invocation
   recovery or appropriate operator intervention.

Why this is an approved, fail-closed design trade-off: the
invocation binding on a pending row is the authorization
boundary that keeps one tenant's (or one workflow run's)
uncommitted work from being claimed by another. Silently
adopting the old pending effect under a new BusinessOpID
would launder that boundary — the retrying invocation never
proved authority over the original operation — so the
framework refuses instead of guessing.

Supported recovery path: retry under the SAME BusinessOpID
from the owning invocation lineage (same-key retry adopts or
reconciles per the C6 semantics above), or operator
intervention (inspect the stuck row, settle it explicitly).
What developers must understand: minting a fresh BusinessOpID
to "route around" a stuck pending effect does NOT unblock the
business operation — it creates a second, independent
operation while the original row stays bound and pending. If
a retry needs to escape a wedged operation, that is an
operator decision with explicit settlement, not a naming
trick in miniapp code.

Status: analyzed but NOT execution-tested. Prompt29
explicitly classifies this wedge as analyzed-only; no crash
test covers the claim-to-authorization window with a
cross-invocation retry, and this paragraph claims none.
