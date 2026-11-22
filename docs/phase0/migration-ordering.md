# Migration Ordering and Ownership (Phase 0, A14)

Three schema owners share one Neon primary: the application (`app`
schema and friends), DBOS (system tables, default `dbos` schema),
and River (job tables, `river` schema).

## Rules

1. Application migrations are owned by this repo under `migrations/`,
   applied by golang-migrate v4.20.1 (see version register).
2. DBOS- and River-managed schemas migrate through their own
   library mechanisms at pinned versions, never through app SQL.
3. Deployment order per release: library-managed migrations first
   (DBOS, then River), then application migrations, then code rollout.
4. CI and deploy gates refuse a release when any owner reports a
   pending or dirty migration state.
5. Application migrations never race library-managed migrations:
   a single ordered gate, one writer at a time.

The `migrations/` directory holds only the skeleton until Phase 1
creates the first application schema.
