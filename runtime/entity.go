package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PithomLabs/miniapp/contract"
)

// MaxEntityLocalLen bounds a tenant-local entity ID. Scoped keys feed
// database uniqueness constraints and queue partition keys; the bound
// keeps them indexable and log-safe.
const MaxEntityLocalLen = 128

// resolveEntity derives the tenant-scoped entity key for one
// invocation: the manifest's input field path resolved against the
// validated input, prefixed with the invoking tenant. Empty
// declaration yields empty scope (no entity). Missing, non-string,
// empty, or overlong values fail closed: a consequential tool must
// never run with an entity it cannot name.
func resolveEntity(decl contract.EntityKey, input json.RawMessage, tenantID string) (string, error) {
	if decl == "" {
		return "", nil
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		return "", fmt.Errorf("entity %q: input is not an object", decl)
	}
	var cur any = doc
	for _, seg := range strings.Split(string(decl), ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", fmt.Errorf("entity %q: path does not resolve", decl)
		}
		cur, ok = m[seg]
		if !ok {
			return "", fmt.Errorf("entity %q: field is missing", decl)
		}
	}
	local, ok := cur.(string)
	if !ok || local == "" {
		return "", fmt.Errorf("entity %q: field must be a non-empty string", decl)
	}
	if len(local) > MaxEntityLocalLen {
		return "", fmt.Errorf("entity %q: value exceeds %d characters", decl, MaxEntityLocalLen)
	}
	return contract.ScopeID(tenantID, local), nil
}
