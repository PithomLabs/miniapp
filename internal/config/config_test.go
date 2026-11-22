package config

import (
	"os"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	setOrUnset(t, "MINIAPP_POLICY", nil)
	setOrUnset(t, "CI", nil)
	if got := LoadPolicy(); got != PolicyWarn {
		t.Fatalf("default = %s, want warn", got)
	}
	t.Setenv("CI", "true")
	if got := LoadPolicy(); got != PolicyStrict {
		t.Fatalf("CI default = %s, want strict", got)
	}
	t.Setenv("MINIAPP_POLICY", "bogus")
	if got := LoadPolicy(); got != PolicyStrict {
		t.Fatalf("unknown = %s, want strict (fail closed)", got)
	}
	t.Setenv("MINIAPP_POLICY", "disabled")
	t.Setenv("CI", "")
	if got := LoadPolicy(); got != PolicyDisabled {
		t.Fatalf("explicit = %s, want disabled", got)
	}
}

// setOrUnset assigns an env var to a value, or unsets it when val is
// nil, restoring the original state after the test.
func setOrUnset(t *testing.T, key string, val *string) {
	t.Helper()
	if val == nil {
		orig, had := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if had {
				os.Setenv(key, orig)
			}
		})
		return
	}
	t.Setenv(key, *val)
}

func strptr(s string) *string { return &s }

// TestLoadPolicyMatrix covers every policy value with CI set and
// unset. Unknown, empty, and malformed values must fail closed to
// strict in both modes; only a genuinely unset variable takes the
// CI-dependent default.
func TestLoadPolicyMatrix(t *testing.T) {
	values := []struct {
		name string
		val  *string
	}{
		{"unset", nil},
		{"strict", strptr("strict")},
		{"warn", strptr("warn")},
		{"disabled", strptr("disabled")},
		{"bogus", strptr("bogus")},
		{"uppercase", strptr("STRICT")},
		{"trailing-space", strptr("strict ")},
		{"empty", strptr("")},
	}
	ciModes := []struct {
		name string
		val  *string
	}{
		{"ci-unset", nil},
		{"ci-true", strptr("true")},
	}
	want := func(val, ci *string) PolicyMode {
		v := ""
		set := val != nil
		if set {
			v = *val
		}
		switch v {
		case "strict":
			return PolicyStrict
		case "warn":
			return PolicyWarn
		case "disabled":
			return PolicyDisabled
		}
		if !set && (ci == nil || *ci != "true") {
			return PolicyWarn
		}
		return PolicyStrict
	}
	for _, c := range ciModes {
		for _, v := range values {
			t.Run(c.name+"/"+v.name, func(t *testing.T) {
				setOrUnset(t, "MINIAPP_POLICY", v.val)
				setOrUnset(t, "CI", c.val)
				if got := LoadPolicy(); got != want(v.val, c.val) {
					t.Fatalf("LoadPolicy() = %s, want %s", got, want(v.val, c.val))
				}
			})
		}
	}
}
