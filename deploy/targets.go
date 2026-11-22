package deploy

import (
	"context"
	"time"

	"github.com/PithomLabs/miniapp/contract"
)

// TaskRequest is the frozen bounded-task declaration a DBOS step hands
// to the runner. Files are capped non-credential inputs; external
// operations execute through the capability proxy (A1).
type TaskRequest struct {
	TaskID        string
	ParentRunID   string
	Snapshot      string
	Command       []string
	Files         map[string][]byte
	Authority     contract.TaskAuthority
	NetworkPolicy contract.SpritePolicy
	Timeout       time.Duration
	ResultRef     string
}

// TaskResult is the terminal outcome copied to Neon before teardown.
type TaskResult struct {
	TaskID   string
	Status   contract.TaskState
	Result   []byte
	ErrorMsg string
}

// SpriteTasks runs bounded isolated tasks for hybrid flows.
// Implemented in Phase 5; the interface is frozen in Phase 0.
type SpriteTasks interface {
	RunTask(ctx context.Context, task TaskRequest) (TaskResult, error)
	GetTaskStatus(ctx context.Context, taskID string) (contract.TaskRecord, error)
	CancelTask(ctx context.Context, taskID string, reason string) error
}

// WorkspaceSpec describes a development or preview workspace.
type WorkspaceSpec struct {
	Name         string
	Snapshot     string
	Profile      contract.DeployProfile
	LeaseTimeout time.Duration
}

// SpriteWorkspaces provisions development and preview workspaces.
// Implemented in Phase 4; the interface is frozen in Phase 0.
type SpriteWorkspaces interface {
	ProvisionWorkspace(ctx context.Context, spec WorkspaceSpec) (string, error)
	DecommissionWorkspace(ctx context.Context, id string) error
	Snapshot(ctx context.Context, id, comment string) (string, error)
	Restore(ctx context.Context, id string, checkpoint string) error
}

// MachineTarget deploys immutable artifacts to Fly Machines.
// Implemented in Phase 5; the interface is frozen in Phase 0.
type MachineTarget interface {
	Deploy(ctx context.Context, artifact contract.PromotionRecord) error
	Status(ctx context.Context, target string) (string, error)
}

// Promotion coordinates publish, promote, and rollback records.
// Publish records land in Phase 4; promote/rollback in Phase 5.
type Promotion interface {
	Publish(ctx context.Context, miniappID string, version contract.Version) (contract.PromotionRecord, error)
	Promote(ctx context.Context, artifact contract.PromotionRecord, target string) error
	Rollback(ctx context.Context, miniappID string, toVersion contract.Version) error
}
