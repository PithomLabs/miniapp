# migrations/ (skeleton)

Application-schema migrations land here starting in Phase 1, applied
by golang-migrate v4.20.1 per `docs/phase0/migration-ordering.md`.

DBOS- and River-managed schemas migrate through their own library
mechanisms at pinned versions — never through SQL in this directory.
