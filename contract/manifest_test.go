package contract

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func loadValidManifest(t *testing.T) MiniappManifest {
	t.Helper()
	data, err := os.ReadFile("testdata/valid-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var mf MiniappManifest
	if err := json.Unmarshal(data, &mf); err != nil {
		t.Fatal(err)
	}
	return mf
}

func TestValidManifestDecodesAndValidates(t *testing.T) {
	mf := loadValidManifest(t)
	if err := mf.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
}

func TestManifestValidationRejects(t *testing.T) {
	base := loadValidManifest(t)
	cases := map[string]func(*MiniappManifest){
		"empty id":            func(m *MiniappManifest) { m.ID = "" },
		"bad version":         func(m *MiniappManifest) { m.Version = "1.0" },
		"no tools":            func(m *MiniappManifest) { m.Tools = nil },
		"bad tool name":       func(m *MiniappManifest) { m.Tools[0].Name = "Nope" },
		"bad class":           func(m *MiniappManifest) { m.Tools[0].Class = "E" },
		"class substrate lie": func(m *MiniappManifest) { m.Tools[0].Substrate = SubstrateDBOS },
		"bad input schema":    func(m *MiniappManifest) { m.Tools[0].InputSchema = json.RawMessage(`[1,2`) },
		"bad trust mode":      func(m *MiniappManifest) { m.Tools[0].TrustModes = []TrustMode{"yolo"} },
		"missing placement":   func(m *MiniappManifest) { delete(m.Deployment.Tools, "rpa.crm.read_contact") },
		"bad placement": func(m *MiniappManifest) {
			d := m.Deployment.Tools["rpa.crm.read_contact"]
			d.Placement = "moon"
			m.Deployment.Tools["rpa.crm.read_contact"] = d
		},
		"unbound profile": func(m *MiniappManifest) {
			d := m.Deployment.Tools["rpa.crm.read_contact"]
			d.Profile.Digest = ""
			m.Deployment.Tools["rpa.crm.read_contact"] = d
		},
		"stray placement": func(m *MiniappManifest) {
			m.Deployment.Tools["rpa.other.tool"] = m.Deployment.Tools["rpa.crm.read_contact"]
		},
	}
	for name, mutate := range cases {
		mf := base
		mf.Tools = append([]ToolManifest(nil), base.Tools...)
		mf.Deployment.Tools = map[string]ToolDeployment{}
		for k, v := range base.Deployment.Tools {
			mf.Deployment.Tools[k] = v
		}
		mutate(&mf)
		if err := mf.Validate(); err == nil {
			t.Errorf("%s: expected rejection, got accept", name)
		} else if !strings.Contains(err.Error(), "terminal") {
			t.Errorf("%s: expected terminal error, got %v", name, err)
		}
	}
}

func TestDuplicateToolRejected(t *testing.T) {
	mf := loadValidManifest(t)
	mf.Tools = append(mf.Tools, mf.Tools[0])
	if err := mf.Validate(); err == nil {
		t.Fatal("expected duplicate tool rejection")
	}
}

func TestScalarValidation(t *testing.T) {
	if err := Name("rpa.crm.read").Validate("f"); err != nil {
		t.Errorf("good name rejected: %v", err)
	}
	if err := Name("BAD NAME").Validate("f"); err == nil {
		t.Error("bad name accepted")
	}
	if err := Version("1.4.2").Validate("f"); err != nil {
		t.Errorf("good version rejected: %v", err)
	}
	if err := Version("v1.4").Validate("f"); err == nil {
		t.Error("bad version accepted")
	}
	good := Digest("sha256:" + strings.Repeat("ab", 32))
	if err := good.Validate("f"); err != nil {
		t.Errorf("good digest rejected: %v", err)
	}
	if err := Digest("sha256:xyz").Validate("f"); err == nil {
		t.Error("bad digest accepted")
	}
}

func TestCapabilityValidation(t *testing.T) {
	good := CapabilitySpec{
		Provider: "crm", Operation: "get_contact", APIVersion: "v3",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		AuthMethod:   "oauth", ConnectionRef: "conn_crm_prod",
		Risk: RiskRead, ReadOnly: true,
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("good capability rejected: %v", err)
	}
	write := good
	write.ReadOnly = false
	write.Risk = RiskConsequential
	if err := write.Validate(); err == nil {
		t.Fatal("write without idempotency accepted")
	}
	write.Idempotency.Required = true
	if err := write.Validate(); err != nil {
		t.Fatalf("idempotent write rejected: %v", err)
	}
}

func TestEffectToolRequiresEntityKeyOrJustification(t *testing.T) {
	mf := loadValidManifest(t)
	mf.Tools[0].Effects = []string{"crm.update_contact"}
	if err := mf.Validate(); err == nil {
		t.Fatal("effect tool without entityKey accepted")
	}
	mf.Tools[0].EntityKey = "contact_id"
	if err := mf.Validate(); err != nil {
		t.Fatalf("effect tool with entityKey rejected: %v", err)
	}
	mf.Tools[0].EntityKey = ""
	mf.Tools[0].NoEntityWhy = "broadcast notification has no entity scope (reviewed)"
	if err := mf.Validate(); err != nil {
		t.Fatalf("effect tool with reviewed justification rejected: %v", err)
	}
}

func TestNegativeLimitsRejected(t *testing.T) {
	mf := loadValidManifest(t)
	mf.Tools[0].Limits.MaxRecordsPerRun = -1
	if err := mf.Validate(); err == nil {
		t.Fatal("negative blast-radius limit accepted")
	}
}

func TestCapabilityAttributionAndOrigin(t *testing.T) {
	good := CapabilitySpec{
		Provider: "crm", Operation: "get_contact", APIVersion: "v3",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		AuthMethod:   "oauth", ConnectionRef: "conn_crm_prod",
		Risk: RiskRead, ReadOnly: true, Attribution: AttributionUser,
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("good capability rejected: %v", err)
	}
	bad := good
	bad.Attribution = "root"
	if err := bad.Validate(); err == nil {
		t.Fatal("bad attribution accepted")
	}
	origin := good
	origin.Origin = "https://api.example.com"
	if err := origin.Validate(); err == nil {
		t.Fatal("custom origin without operator approver accepted")
	}
	origin.OriginApprovedBy = "sec-owner"
	if err := origin.Validate(); err != nil {
		t.Fatalf("approved origin rejected: %v", err)
	}
	origin.Origin = "http://api.example.com/hook?x=1"
	if err := origin.Validate(); err == nil {
		t.Fatal("non-https origin with path accepted")
	}
}

func TestCallerValidation(t *testing.T) {
	good := Caller{AgentID: "codex-1", Authority: Authority{Kind: AuthorityUser, ID: "u1", TenantID: "t1"}}
	if err := good.Validate(); err != nil {
		t.Fatalf("good caller rejected: %v", err)
	}
	bad := good
	bad.Authority.TenantID = ""
	if err := bad.Validate(); err == nil {
		t.Fatal("caller without tenant accepted")
	}
}
