package runtime

import "fmt"

import "github.com/PithomLabs/miniapp/contract"

// Error is a typed invocation failure. Kind follows the shared
// taxonomy: terminal failures must not be retried unchanged,
// retryable failures may be, quarantined outcomes need an operator.
type Error struct {
	Kind contract.ErrorKind
	Code string
	Msg  string
}

func (e *Error) Error() string { return string(e.Kind) + " " + e.Code + ": " + e.Msg }

func terminal(code, format string, args ...any) *Error {
	return &Error{Kind: contract.KindTerminal, Code: code, Msg: fmt.Sprintf(format, args...)}
}

func retryable(code, format string, args ...any) *Error {
	return &Error{Kind: contract.KindRetryable, Code: code, Msg: fmt.Sprintf(format, args...)}
}
