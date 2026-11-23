package testkit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
)

// ClassAManifest builds a valid single-tool Class A manifest for
// tests. entityKey is the input field path ("" for none); effects
// lists declared external effects. The manifest passes structural
// validation; registry admission is the test's to attempt.
func ClassAManifest(id, toolName, entityKey string, effects []string) *contract.MiniappManifest {
	sum := sha256.Sum256([]byte(`{"name":"default","placement":"machine","substrate":"go"}`))
	profile := contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
	schema := json.RawMessage(`{"type":"object"}`)
	tool := contract.ToolManifest{
		Name: contract.ToolName(toolName), Description: "Test tool.",
		Class: contract.ClassA, Substrate: contract.SubstrateGo,
		InputSchema: schema, OutputSchema: schema,
		TrustModes: []contract.TrustMode{contract.TrustAutonomous},
		EntityKey:  contract.EntityKey(entityKey),
		Effects:    effects,
	}
	if entityKey == "" && len(effects) == 0 {
		tool.NoEntityWhy = "Test tool without effects."
	}
	return &contract.MiniappManifest{
		ID: contract.Name(id), Version: contract.Version("1.0.0"),
		Description: "Test miniapp.",
		Tools:       []contract.ToolManifest{tool},
		Deployment: contract.DeploymentSection{Tools: map[string]contract.ToolDeployment{
			toolName: {Placement: contract.PlacementMachine,
				Profile: contract.ProfileRef{Name: "default", Digest: profile}},
		}},
	}
}

// ClassAImpl binds one handler to a test tool.
func ClassAImpl(build, toolName string, handler registry.ToolFunc) *registry.Implementation {
	return &registry.Implementation{Build: build,
		ClassATools: map[string]registry.ToolFunc{toolName: handler}}
}

// Envelope binds a test manifest digest to a build identity.
func Envelope(t *testing.T, mf *contract.MiniappManifest, build string) contract.ReleaseEnvelope {
	t.Helper()
	digest, err := contract.ProfileDigest(mf)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ReleaseEnvelope{Manifest: digest, Implementation: build,
		Build: build, SourceRevision: "test", Artifact: digest}
}

// RegisterClassATool builds and admits a single-tool Class A test
// miniapp. Use ClassAManifest + reg.Register directly when the test
// asserts admission failure.
func RegisterClassATool(t *testing.T, reg *registry.Registry, id, toolName, entityKey string, effects []string, handler registry.ToolFunc) {
	t.Helper()
	mf := ClassAManifest(id, toolName, entityKey, effects)
	if err := reg.Register(mf, ClassAImpl("test", toolName, handler), Envelope(t, mf, "test")); err != nil {
		t.Fatalf("register %s: %v", id, err)
	}
}
