# Retention Requirements (Phase 0; policy artifact in Phase 4, A10)

Active runs never expire arbitrarily; terminal and operational records
are bounded. The Phase 4 policy artifact must define, per record class,
retention window, archival destination and format, deletion behavior,
owner, redaction rules, access controls, and legal/security constraints.

## Record classes in scope

Active runs, terminal run history, DBOS workflow history, River jobs,
effect attempt/outcome records, audit events, Sprite task records,
promotion records, service logs, and distributed traces.

## Requirements

1. No unbounded growth: every class has a window or quota with an
   enforced deletion/archive job and an owner.
2. Audit events outlive the runs they describe; deletion needs a
   documented basis and approval.
3. Archived records remain queryable for incident review within a
   stated time bound.
4. Retention configuration and current usage are visible in the
   operations view (§17).
5. Redaction rules apply before archival; secrets never persist in
   terminal records beyond their operational need.
