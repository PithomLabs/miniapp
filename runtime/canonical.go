package runtime

import (
	"bytes"
	"encoding/json"

	"github.com/PithomLabs/miniapp/contract"
)

// canonicalJSON delegates to the shared contract implementation
// (single definition; this wrapper preserves existing call sites).
func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	return contract.CanonicalJSON(raw)
}

// sameInput reports whether two inputs are semantically identical.
// Unparseable inputs are never equal.
func sameInput(a, b json.RawMessage) bool {
	ca, err := canonicalJSON(a)
	if err != nil {
		return false
	}
	cb, err := canonicalJSON(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ca, cb)
}
