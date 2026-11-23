package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"
)

// CredentialHeader carries the caller's API credential. It travels
// in a header (never a query string) so URLs stay log-safe.
const CredentialHeader = "X-Miniapp-Credential"

// Server adapts one canonical service to HTTP.
type Server struct {
	svc *runtime.Service
}

// New returns a Server over svc.
func New(svc *runtime.Service) *Server {
	return &Server{svc: svc}
}

type invokeBody struct {
	MiniappID      string             `json:"miniapp_id"`
	Version        string             `json:"version"`
	Tool           string             `json:"tool"`
	Input          json.RawMessage    `json:"input"`
	IdempotencyKey string             `json:"idempotency_key"`
	RequestedTrust contract.TrustMode `json:"requested_trust"`
}

type signalBody struct {
	RunID   string          `json:"run_id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type cancelBody struct {
	RunID  string `json:"run_id"`
	Reason string `json:"reason"`
}

type decideBody struct {
	ApprovalID string                    `json:"approval_id"`
	Decision   contract.ApprovalDecision `json:"decision"`
	IntentHash contract.Digest           `json:"intent_hash"`
}

type errorBody struct {
	Kind    string `json:"kind"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Handler returns the route set. Every route authenticates through
// the service; transport-level failures (bad JSON, wrong method) are
// the only errors born here.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/invoke", s.handleInvoke)
	mux.HandleFunc("GET /v1/status", s.handleStatus)
	mux.HandleFunc("POST /v1/signal", s.handleSignal)
	mux.HandleFunc("POST /v1/cancel", s.handleCancel)
	mux.HandleFunc("POST /v1/decide", s.handleDecide)
	return mux
}

func credential(r *http.Request) string {
	return r.Header.Get(CredentialHeader)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var re *runtime.Error
	if !errors.As(err, &re) {
		writeJSON(w, http.StatusInternalServerError, errorBody{Kind: "internal", Code: "internal", Message: "internal error"})
		return
	}
	writeJSON(w, statusFor(re), errorBody{Kind: string(re.Kind), Code: re.Code, Message: re.Msg})
}

// statusFor maps the service's typed errors to HTTP statuses. The
// mapping is mechanical: authentication, authorization, identity,
// conflict, and availability keep their standard meanings.
func statusFor(re *runtime.Error) int {
	if re.Kind == contract.KindRetryable {
		return http.StatusServiceUnavailable
	}
	switch re.Code {
	case "auth-required":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "unknown-miniapp", "unknown-tool", "unknown-invocation", "unknown-run", "unknown-approval":
		return http.StatusNotFound
	case "idempotency-key-reuse", "execution-in-progress", "effect-contention":
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	var body invokeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Kind: "terminal", Code: "malformed-body", Message: "request body is not valid JSON"})
		return
	}
	out, err := s.svc.Invoke(r.Context(), runtime.InvokeRequest{
		MiniappID: body.MiniappID, Version: body.Version, Tool: body.Tool,
		Input: body.Input, IdempotencyKey: body.IdempotencyKey,
		Credential: credential(r), RequestedTrust: body.RequestedTrust,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("invocation_id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Kind: "terminal", Code: "missing-invocation-id", Message: "invocation_id is required"})
		return
	}
	out, err := s.svc.Status(r.Context(), credential(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSignal(w http.ResponseWriter, r *http.Request) {
	var body signalBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Kind: "terminal", Code: "malformed-body", Message: "request body is not valid JSON"})
		return
	}
	sig := contract.Signal{Type: body.Type, Payload: body.Payload}
	if err := s.svc.Signal(r.Context(), credential(r), body.RunID, sig); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "delivered"})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	var body cancelBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Kind: "terminal", Code: "malformed-body", Message: "request body is not valid JSON"})
		return
	}
	res, err := s.svc.Cancel(r.Context(), credential(r), body.RunID, body.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleDecide(w http.ResponseWriter, r *http.Request) {
	var body decideBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Kind: "terminal", Code: "malformed-body", Message: "request body is not valid JSON"})
		return
	}
	if err := s.svc.DecideApproval(r.Context(), credential(r), body.ApprovalID, body.Decision, body.IntentHash); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "decided"})
}
