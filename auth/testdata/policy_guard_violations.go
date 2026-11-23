// Planted policy-guard violations: every marked line must be
// flagged by TestP23GuardNegativeControl. Never built (testdata);
// must parse, need not type-check.
package guardtestdata

import "github.com/PithomLabs/miniapp/auth"

func Planted(p *auth.Policy) {
	_ = p.Grants["t"]    // want: subscript read
	p.Grants = nil       // want: whole-map assignment
	p.Grants["t"] = nil  // want: element assignment
	_ = auth.Policy{}    // want: bare literal
	delete(p.Keys, "k")  // want: delete
	for range p.Grants { // want: range
		break
	}
	_ = len(p.Keys) // want: len
}
