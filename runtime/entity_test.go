package runtime

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
)

// TestResolveEntityCollision is the X-19 binding discriminator:
// tenant `a` + local `b:c` and tenant `a:b` + local `c` produced
// the same raw join; the canonical encoding keeps them distinct.
func TestResolveEntityCollision(t *testing.T) {
	a, err := resolveEntity("subject", json.RawMessage(`{"subject":"b:c"}`), "a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := resolveEntity("subject", json.RawMessage(`{"subject":"c"}`), "a:b")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("collision: %q == %q", a, b)
	}
}

func TestResolveEntity(t *testing.T) {
	long := strings.Repeat("x", MaxEntityLocalLen+1)
	cases := []struct {
		name  string
		decl  string
		input string
		want  string
		fail  bool
	}{
		{"empty decl", "", `{"a":"b"}`, "", false},
		{"flat field", "lead_id", `{"lead_id":"l1"}`, "tenant-a:l1", false},
		{"nested path", "lead.contact_id", `{"lead":{"contact_id":"c9"}}`, "tenant-a:c9", false},
		{"missing field", "lead_id", `{"other":"x"}`, "", true},
		{"non-string", "lead_id", `{"lead_id":42}`, "", true},
		{"empty string", "lead_id", `{"lead_id":""}`, "", true},
		{"null", "lead_id", `{"lead_id":null}`, "", true},
		{"path through scalar", "lead.contact_id", `{"lead":"x"}`, "", true},
		{"overlong", "lead_id", `{"lead_id":"` + long + `"}`, "", true},
		{"non-object", "lead_id", `[1,2]`, "", true},
		{"bad json", "lead_id", `{oops`, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveEntity(contract.EntityKey(c.decl), json.RawMessage(c.input), "tenant-a")
			if c.fail {
				if err == nil {
					t.Fatalf("resolveEntity(%q, %s) accepted, want failure", c.decl, c.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveEntity: %v", err)
			}
			if got != c.want {
				t.Fatalf("resolveEntity = %q, want %q", got, c.want)
			}
		})
	}
}
