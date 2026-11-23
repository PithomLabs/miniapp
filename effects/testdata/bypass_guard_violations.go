// Planted provider-Send guard violations: every marked line must
// be flagged by TestP28GuardNegativeControl, and no unmarked
// line may be flagged. Never built (testdata); must parse, need
// not type-check.
package guardtestdata

type rogueChannel struct{}

func (rogueChannel) Send() {} // want: Send implementation

type plantedProbe struct{ ch rogueChannel }

func Planted(p plantedProbe) {
	p.ch.Send()   // want: Send call site
	dbos.Send()   // allowlisted SDK workflow signaling: must stay silent
	_ = p.ch.Send // Send selector without a call: stays silent
}
