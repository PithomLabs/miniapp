# Data Classification, Payload, and Erasure Rules (Phase 0, F20)

## Classification

- Identifiers and compact execution context: allowed in workflow
  inputs, step payloads, and checkpoints.
- PII and customer content: forbidden in DBOS payloads and
  checkpoints; stored only in application tables with a deletion path.
- Credentials and tokens: forbidden everywhere except the encrypted
  connection registry; references only elsewhere.

## Consequences

- Workflow payloads are IDs-only by construction; reviewers reject
  customer records, payloads snapshots, tokens, or secrets in any
  workflow input/output.
- Checkpoint retention never becomes an uncontrolled PII copy.
- Erasure requests remove or irreversibly de-identify eligible
  application records; only legally permitted audit evidence survives.
  End-to-end erasure tests land before production GA.

See also: retention requirements (per-class windows) and the Phase 6
retention/erasure drills.
