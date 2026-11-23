package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// CanonicalJSON normalizes a JSON document so semantically
// identical inputs compare equal regardless of whitespace or key
// order. encoding/json emits map keys sorted, which makes the
// re-marshaled form canonical for comparison and hashing.
func CanonicalJSON(raw json.RawMessage) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// IntentHash binds an operation to its exact arguments. Agents and
// deciders recompute it identically; any material change produces a
// different hash and invalidates outstanding approvals. Input is
// canonicalized first so insignificant serialization differences
// (whitespace, key order) never invalidate an approval.
//
// Lives here (not runtime) so both the submission choke and the
// effect-authorization boundary recompute it from authoritative
// row state without an import cycle.
func IntentHash(miniapp, version, tool string, input json.RawMessage) Digest {
	canonical, err := CanonicalJSON(input)
	if err != nil {
		canonical = input
	}
	sum := sha256.Sum256([]byte(miniapp + "\x00" + version + "\x00" + tool + "\x00" + string(canonical)))
	return Digest(fmt.Sprintf("sha256:%x", sum))
}
