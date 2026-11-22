package config

import "os"

// PolicyMode selects validation strictness: strict fails closed in CI
// and production, warn logs in development, and disabled skips checks
// in controlled tests only.
type PolicyMode string

const (
	PolicyStrict   PolicyMode = "strict"
	PolicyWarn     PolicyMode = "warn"
	PolicyDisabled PolicyMode = "disabled"
)

// LoadPolicy reads MINIAPP_POLICY. Recognized values keep their meaning
// in every environment. An unset variable defaults to warn, or strict
// under CI. Any other value — unknown, empty, or malformed — fails
// closed to strict regardless of CI, so a typo can never silently
// weaken validation.
func LoadPolicy() PolicyMode {
	v, set := os.LookupEnv("MINIAPP_POLICY")
	switch v {
	case "strict":
		return PolicyStrict
	case "warn":
		return PolicyWarn
	case "disabled":
		return PolicyDisabled
	default:
		if !set {
			if os.Getenv("CI") == "true" {
				return PolicyStrict
			}
			return PolicyWarn
		}
		return PolicyStrict
	}
}
