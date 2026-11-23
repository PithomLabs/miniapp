package runtime

import (
	"bytes"
	"encoding/json"
)

// canonicalJSON normalizes a JSON document so semantically identical
// inputs compare equal regardless of whitespace or key order.
// encoding/json emits map keys sorted, which makes the re-marshaled
// form canonical for comparison and hashing.
func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
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
