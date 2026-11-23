package crmread

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
)

// Identity of the reference miniapp.
const (
	ID      = "crm-read"
	Version = "1.0.0"
	// Build identifies this development implementation. Production
	// releases stamp the OCI artifact digest here instead.
	Build = "local-dev"
	// ToolSearch is the single read-only tool.
	ToolSearch = "crm.search_contacts"
)

// profileDoc is the deployment profile the manifest binds by digest.
// Phase 1 serves Class A tools in-process on the host machine; the
// digest pins that placement decision to the manifest.
const profileDoc = `{"name":"default","placement":"machine","substrate":"go"}`

func profileDigest() contract.Digest {
	sum := sha256.Sum256([]byte(profileDoc))
	return contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

var inputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"query": {"type": "string", "description": "Substring matched against name and email. Empty matches all contacts."},
		"limit": {"type": "integer", "minimum": 1, "maximum": 100, "description": "Max contacts returned. Default 20."}
	}
}`)

var outputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"contacts": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"id": {"type": "string"},
					"name": {"type": "string"},
					"email": {"type": "string"}
				},
				"required": ["id", "name", "email"]
			}
		}
	},
	"required": ["contacts"]
}`)

// Manifest returns the versioned artifact of record.
func Manifest() *contract.MiniappManifest {
	return &contract.MiniappManifest{
		ID:          contract.Name(ID),
		Version:     contract.Version(Version),
		Description: "Read-only reference miniapp: search CRM contacts. No external effects.",
		Tools: []contract.ToolManifest{
			{
				Name:         contract.ToolName(ToolSearch),
				Description:  "Search contacts by name or email substring within the caller's tenant.",
				Class:        contract.ClassA,
				Substrate:    contract.SubstrateGo,
				InputSchema:  inputSchema,
				OutputSchema: outputSchema,
				TrustModes:   []contract.TrustMode{contract.TrustConfirm, contract.TrustAutonomous},
				NoEntityWhy:  "Read-only tool: no external effects, no entity serialization needed.",
				Timeout:      "30s",
			},
		},
		Deployment: contract.DeploymentSection{
			Tools: map[string]contract.ToolDeployment{
				ToolSearch: {
					Placement: contract.PlacementMachine,
					Profile:   contract.ProfileRef{Name: "default", Digest: profileDigest()},
				},
			},
			Promotion: contract.PromotionPolicy{ReviewRequired: true},
		},
		Dependencies: contract.DependencyManifest{Policy: "phase1-dev"},
	}
}

// Implementation returns the executable side: one Class A handler.
func Implementation() *registry.Implementation {
	return &registry.Implementation{
		Build:       Build,
		ClassATools: map[string]registry.ToolFunc{ToolSearch: searchContacts},
	}
}

// Envelope binds this manifest digest to this implementation.
func Envelope() (contract.ReleaseEnvelope, error) {
	digest, err := contract.ProfileDigest(Manifest())
	if err != nil {
		return contract.ReleaseEnvelope{}, err
	}
	artifact, err := contract.ProfileDigest(Build)
	if err != nil {
		return contract.ReleaseEnvelope{}, err
	}
	return contract.ReleaseEnvelope{
		Manifest:       digest,
		Implementation: Build,
		Build:          Build,
		SourceRevision: "working-tree",
		Artifact:       artifact,
	}, nil
}

// Register admits crm-read to reg.
func Register(reg *registry.Registry) error {
	env, err := Envelope()
	if err != nil {
		return err
	}
	return reg.Register(Manifest(), Implementation(), env)
}

type searchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type contact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// searchContacts runs inside the caller's tenant transaction. It
// predicates on the tenant explicitly; the table RLS policy enforces
// the same boundary at the database layer.
func searchContacts(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	var in searchInput
	if len(call.Input) > 0 {
		if err := json.Unmarshal(call.Input, &in); err != nil {
			return nil, fmt.Errorf("crm-read: invalid input: %w", err)
		}
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := call.Tx.Query(ctx, `SELECT id, name, email FROM app.crm_contacts
		WHERE tenant_id = $1 AND (name ILIKE $2 OR email ILIKE $2)
		ORDER BY name LIMIT $3`, call.TenantID, "%"+in.Query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := struct {
		Contacts []contact `json:"contacts"`
	}{Contacts: []contact{}}
	for rows.Next() {
		var c contact
		if err := rows.Scan(&c.ID, &c.Name, &c.Email); err != nil {
			return nil, err
		}
		out.Contacts = append(out.Contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}
