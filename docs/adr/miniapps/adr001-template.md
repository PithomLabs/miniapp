# ADR-001 decision — <miniapp id> v<version>

Required for every Class C/D miniapp before its substrate is enabled.
Automated admission checks presence, score arithmetic, approved state,
and test links; the business-consequence and net-complexity judgments
require the named human reviewer below.

- Business promise: <what must never break>
- Score (0/1/2): lifetime _  stages _  waiting _  recovery _  effects _  coordination _  human _  = __/14
- Gate 1 durable need: <which requirement>
- Gate 2 characteristics (need ≥2): <list>
- Gate 3 failure consequence: <text>
- Alternative A (without DBOS) custom machinery: <state, timers, reaper, ...>
- Alternative B (DBOS) covers / still needs us to build: <lists>
- §7 checks (link each to a test): determinism lint | idempotency keys |
  tx boundary | workflow ID | versioning plan | payload discipline |
  recovery policy | op cost
- Custom machinery estimate (Alternative A): <work>
- Residual work under Alternative B: <work>
- Decision: REJECT | DEFER | PILOT | ADOPT
- Reviewer (human): <name>   Date: <date>
