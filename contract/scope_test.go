package contract

import "testing"

// TestScopeIDCollisionPair is the X-19 discriminator: tenant `a`
// with local `b:c` and tenant `a:b` with local `c` produced the
// same raw `tenant:local` string; the canonical encoding keeps
// them distinct.
func TestScopeIDCollisionPair(t *testing.T) {
	a := ScopeID("a", "b:c")
	b := ScopeID("a:b", "c")
	if a == b {
		t.Fatalf("collision: %q == %q", a, b)
	}
	if a != "a:b%3Ac" || b != "a%3Ab:c" {
		t.Fatalf("got %q and %q, want a:b%%3Ac and a%%3Ab:c", a, b)
	}
}

// TestScopeIDLegacyIdentity pins backward compatibility: IDs with
// no delimiters encode exactly as the historical raw join, so
// pre-encoding persisted rows remain findable.
func TestScopeIDLegacyIdentity(t *testing.T) {
	for _, tc := range [][3]string{
		{"tenant-a", "order-1", "tenant-a:order-1"},
		{"t", "x", "t:x"},
		{"prod", "sub", "prod:sub"},
	} {
		if got := ScopeID(tc[0], tc[1]); got != tc[2] {
			t.Fatalf("ScopeID(%q,%q) = %q, want %q", tc[0], tc[1], got, tc[2])
		}
	}
	if got := ScopePrefix("tenant-a"); got != "tenant-a:" {
		t.Fatalf("ScopePrefix = %q, want tenant-a:", got)
	}
}

// TestScopeIDEscapePrecedence pins the escape table: `%` escapes
// first so pre-existing escape-looking text can never forge a
// different identity's encoding.
func TestScopeIDEscapePrecedence(t *testing.T) {
	// "a%3A" as raw input must not encode to the same string as
	// raw "a:" — the pre-existing escape-looking text is itself
	// escaped.
	raw := ScopeID("t", "a:")
	crafty := ScopeID("t", "a%3A")
	if raw == crafty {
		t.Fatalf("crafted collision: %q == %q", raw, crafty)
	}
	if raw != "t:a%3A" || crafty != "t:a%253A" {
		t.Fatalf("got %q and %q", raw, crafty)
	}
}
