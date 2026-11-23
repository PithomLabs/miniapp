package mcp

import (
	"context"
	"encoding/json"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"
)

// ServerName identifies this tool set to harness clients.
const ServerName = "miniapp"

// Tool names served by this adapter.
const (
	ToolInvoke = "miniapp_invoke"
	ToolStatus = "runs_status"
	ToolSignal = "runs_signal"
	ToolCancel = "runs_cancel"
	ToolDecide = "approvals_decide"
)

type invokeArgs struct {
	Credential     string             `json:"credential"`
	MiniappID      string             `json:"miniapp_id"`
	Version        string             `json:"version"`
	Tool           string             `json:"tool"`
	Input          map[string]any     `json:"input"`
	IdempotencyKey string             `json:"idempotency_key"`
	RequestedTrust contract.TrustMode `json:"requested_trust"`
}

type statusArgs struct {
	Credential   string `json:"credential"`
	InvocationID string `json:"invocation_id"`
}

type signalArgs struct {
	Credential string          `json:"credential"`
	RunID      string          `json:"run_id"`
	Type       string          `json:"type"`
	Payload    *map[string]any `json:"payload"`
}

type cancelArgs struct {
	Credential string `json:"credential"`
	RunID      string `json:"run_id"`
	Reason     string `json:"reason"`
}

type decideArgs struct {
	Credential string                    `json:"credential"`
	ApprovalID string                    `json:"approval_id"`
	Decision   contract.ApprovalDecision `json:"decision"`
	IntentHash contract.Digest           `json:"intent_hash"`
}

// NewServer builds the MCP tool set over svc.
func NewServer(svc *runtime.Service) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: ServerName, Version: "1.0.0"}, nil)
	sdk.AddTool(server, &sdk.Tool{Name: ToolInvoke, Description: "Invoke a miniapp tool through the governed canonical service."},
		func(ctx context.Context, _ *sdk.CallToolRequest, args invokeArgs) (*sdk.CallToolResult, any, error) {
			input, err := remarshal(args.Input)
			if err != nil {
				return nil, nil, err
			}
			out, err := svc.Invoke(ctx, runtime.InvokeRequest{
				MiniappID: args.MiniappID, Version: args.Version, Tool: args.Tool,
				Input: input, IdempotencyKey: args.IdempotencyKey,
				Credential: args.Credential, RequestedTrust: args.RequestedTrust,
			})
			if err != nil {
				return nil, nil, err
			}
			return jsonResult(out)
		})
	sdk.AddTool(server, &sdk.Tool{Name: ToolStatus, Description: "Report one invocation's lifecycle status."},
		func(ctx context.Context, _ *sdk.CallToolRequest, args statusArgs) (*sdk.CallToolResult, any, error) {
			out, err := svc.Status(ctx, args.Credential, args.InvocationID)
			if err != nil {
				return nil, nil, err
			}
			return jsonResult(out)
		})
	sdk.AddTool(server, &sdk.Tool{Name: ToolSignal, Description: "Deliver an external input to a live run."},
		func(ctx context.Context, _ *sdk.CallToolRequest, args signalArgs) (*sdk.CallToolResult, any, error) {
			var payload json.RawMessage = json.RawMessage(`{}`)
			if args.Payload != nil {
				var err error
				payload, err = remarshal(*args.Payload)
				if err != nil {
					return nil, nil, err
				}
			}
			sig := contract.Signal{Type: args.Type, Payload: payload}
			if err := svc.Signal(ctx, args.Credential, args.RunID, sig); err != nil {
				return nil, nil, err
			}
			return textResult("delivered")
		})
	sdk.AddTool(server, &sdk.Tool{Name: ToolCancel, Description: "Stop a live run and report in-flight effects."},
		func(ctx context.Context, _ *sdk.CallToolRequest, args cancelArgs) (*sdk.CallToolResult, any, error) {
			res, err := svc.Cancel(ctx, args.Credential, args.RunID, args.Reason)
			if err != nil {
				return nil, nil, err
			}
			return jsonResult(res)
		})
	sdk.AddTool(server, &sdk.Tool{Name: ToolDecide, Description: "Record a human decision on a pending approval."},
		func(ctx context.Context, _ *sdk.CallToolRequest, args decideArgs) (*sdk.CallToolResult, any, error) {
			if err := svc.DecideApproval(ctx, args.Credential, args.ApprovalID, args.Decision, args.IntentHash); err != nil {
				return nil, nil, err
			}
			return textResult("decided")
		})
	return server
}

// remarshal converts a decoded free-form argument back to raw JSON.
// Object arguments use map form because the SDK's schema derivation
// maps raw JSON to a boolean schema some clients reject; a nil
// argument becomes an empty object for tools whose input is optional.
// Note: MCP signal payloads are objects; other transports accept any
// JSON value.
func remarshal(v any) (json.RawMessage, error) {
	if v == nil {
		return json.RawMessage(`{}`), nil
	}
	return json.Marshal(v)
}

func jsonResult(v any) (*sdk.CallToolResult, any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: string(raw)}}}, nil, nil
}

func textResult(s string) (*sdk.CallToolResult, any, error) {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: s}}}, nil, nil
}
