package classd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
)

// Identity of the conformance probe.
const (
	ID      = "classd-probe"
	Version = "1.0.0"
	Build   = "test-fixture"

	// ToolRun is approval-gated; ToolOpen is identical except the
	// approval wait is skipped. Both run the same sequence logic.
	ToolRun  = "probe.sequence_run"
	ToolOpen = "probe.sequence_open"

	// WorkflowVersion is the family's compatibility line.
	WorkflowVersion = "1"
)

const profileDoc = `{"name":"default","placement":"machine","substrate":"dbos"}`

func profileDigest() contract.Digest {
	sum := sha256.Sum256([]byte(profileDoc))
	return contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

var inputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"subject_id": {"type": "string"},
		"sleep_ms": {"type": "integer", "minimum": 0},
		"wait_ms": {"type": "integer", "minimum": 0},
		"approval_timeout_ms": {"type": "integer", "minimum": 0},
		"poison_stage": {"type": "integer", "minimum": 0}
	},
	"required": ["subject_id"]
}`)

var outputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"sequence": {"type": "string"},
		"generation": {"type": "integer"},
		"outcome": {"type": "string"},
		"stages": {"type": "integer"}
	},
	"required": ["sequence", "generation", "outcome", "stages"]
}`)

// Manifest returns the probe's test-only manifest.
func Manifest() *contract.MiniappManifest {
	tool := func(name, desc string, approval contract.ApprovalRequirement) contract.ToolManifest {
		return contract.ToolManifest{
			Name: contract.ToolName(name), Description: desc,
			Class: contract.ClassD, Substrate: contract.SubstrateDBOS,
			InputSchema: inputSchema, OutputSchema: outputSchema,
			TrustModes: []contract.TrustMode{contract.TrustConfirm, contract.TrustAutonomous},
			Approval:   approval, EntityKey: "subject_id", Effects: []string{"mock.send"},
			NoEntityWhy: "", Timeout: "10m",
		}
	}
	dep := contract.ToolDeployment{Placement: contract.PlacementMachine,
		Profile: contract.ProfileRef{Name: "default", Digest: profileDigest()}}
	return &contract.MiniappManifest{
		ID: contract.Name(ID), Version: contract.Version(Version),
		Description: "Class D conformance probe (test-only, no business semantics).",
		Tools: []contract.ToolManifest{
			tool(ToolRun, "Run a sequence behind an approval gate.",
				contract.ApprovalRequirement{Required: true, ApproverScope: "tenant-ops", Expiry: "10m"}),
			tool(ToolOpen, "Run a sequence without an approval gate.", contract.ApprovalRequirement{}),
		},
		Deployment: contract.DeploymentSection{Tools: map[string]contract.ToolDeployment{
			ToolRun: dep, ToolOpen: dep,
		}},
		Dependencies: contract.DependencyManifest{Policy: "test-fixture"},
	}
}

// Implementation binds both tools to the shared sequence logic
// through distinct workflow entry points (DBOS registers by
// function; two tools cannot share one registration).
func Implementation() *registry.Implementation {
	return &registry.Implementation{
		Build: Build,
		Workflows: map[string]registry.WorkflowDef{
			ToolRun:  {Name: ToolRun, Version: WorkflowVersion, Register: registerRun, Start: startRun},
			ToolOpen: {Name: ToolOpen, Version: WorkflowVersion, Register: registerOpen, Start: startOpen},
		},
	}
}

// Envelope binds this manifest digest to this implementation.
func Envelope() (contract.ReleaseEnvelope, error) {
	digest, err := contract.ProfileDigest(Manifest())
	if err != nil {
		return contract.ReleaseEnvelope{}, err
	}
	return contract.ReleaseEnvelope{
		Manifest: digest, Implementation: Build, Build: Build,
		SourceRevision: "test-fixture", Artifact: digest,
	}, nil
}

// Register admits the probe to reg.
func Register(reg *registry.Registry) error {
	env, err := Envelope()
	if err != nil {
		return err
	}
	return reg.Register(Manifest(), Implementation(), env)
}
