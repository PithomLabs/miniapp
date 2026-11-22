// Package deploy owns Dimension B: deployment profiles, compute-target
// adapters, bounded Sprite tasks, and promotion records.
//
// Phase 0 scope is profiles plus interfaces only. Implementation
// ownership (A5): Sprite workspaces in Phase 4; the Machines adapter,
// Sprite task runner, REST gap client, and promote/rollback in Phase 5;
// Phase 6 consumes and hardens both adapters.
package deploy
