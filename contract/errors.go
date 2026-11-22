package contract

import "fmt"

// ErrorKind classifies failures per the shared terminal-error taxonomy.
// Retryable and quarantined kinds exist so callers can map contract
// failures into run outcomes without string matching.
type ErrorKind string

const (
	// KindTerminal marks a failure that retry cannot fix (bad manifest,
	// unknown capability, policy rejection).
	KindTerminal ErrorKind = "terminal"
	// KindRetryable marks a transient failure the caller may retry.
	KindRetryable ErrorKind = "retryable"
	// KindQuarantined marks an outcome needing operator reconciliation.
	KindQuarantined ErrorKind = "quarantined"
)

// Error is a typed contract failure. Contract code returns *Error
// values, never bare strings or fmt.Errorf results.
type Error struct {
	Kind  ErrorKind
	Field string
	Msg   string
}

func (e *Error) Error() string {
	if e.Field == "" {
		return string(e.Kind) + ": " + e.Msg
	}
	return string(e.Kind) + " " + e.Field + ": " + e.Msg
}

// Terminal builds a terminal contract error for a field.
func Terminal(field, format string, args ...any) *Error {
	return &Error{Kind: KindTerminal, Field: field, Msg: fmt.Sprintf(format, args...)}
}

// MultiError aggregates field failures so one Validate call reports
// every problem instead of stopping at the first.
type MultiError struct {
	Errs []*Error
}

func (m *MultiError) Error() string {
	if len(m.Errs) == 1 {
		return m.Errs[0].Error()
	}
	return fmt.Sprintf("%d contract errors; first: %s", len(m.Errs), m.Errs[0].Error())
}

// Add appends a non-nil *Error. Nil errors are ignored.
func (m *MultiError) Add(err *Error) {
	if err != nil {
		m.Errs = append(m.Errs, err)
	}
}

// OrNil returns nil when no errors were collected.
func (m *MultiError) OrNil() error {
	if len(m.Errs) == 0 {
		return nil
	}
	return m
}
