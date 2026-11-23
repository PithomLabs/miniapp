package transport_test

// Prompt28 workstream D — /healthz exposes exactly liveness plus
// the grant-configuration hash, unauthenticated, using the same
// canonicalization as auth.Policy (string-identical to a direct
// ConfigHash read). Grant changes are visible on the next poll.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
)

func TestP28HealthzExposesHash(t *testing.T) {
	a := setupAdapters(t)
	// Deliberately no credential header: health is public.
	resp, err := http.Get(a.srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 2 || body["status"] != "ok" {
		t.Fatalf("body = %v, want exactly {status:ok, grant_config_hash}", body)
	}
	hash, ok := body["grant_config_hash"]
	if !ok {
		t.Fatalf("body = %v, want grant_config_hash", body)
	}
	if hash != a.fx.Policy.ConfigHash() {
		t.Errorf("healthz hash = %q, want policy hash %q (same canonicalization)",
			hash, a.fx.Policy.ConfigHash())
	}
	for _, leak := range []string{"tenant", "agent", "token", "secret", "password", "credential"} {
		if strings.Contains(strings.ToLower(hash), leak) {
			t.Errorf("hash %q contains %q; hashes must be credential-free", hash, leak)
		}
	}
	// Sensitivity through the consumer: a grant change flips the
	// next poll.
	a.fx.Policy.SetGrants(map[string]map[string]contract.TrustMode{
		"t-revoked": {"m-revoked": contract.TrustConfirm},
	})
	resp2, err := http.Get(a.srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var body2 map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&body2); err != nil {
		t.Fatal(err)
	}
	if body2["grant_config_hash"] == hash {
		t.Errorf("hash unchanged after grant swap; consumer must track the snapshot")
	}
	if body2["grant_config_hash"] != a.fx.Policy.ConfigHash() {
		t.Errorf("healthz hash = %q, want policy hash %q", body2["grant_config_hash"], a.fx.Policy.ConfigHash())
	}
}
