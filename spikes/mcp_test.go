package spikes

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPRoundTrip proves the pinned MCP SDK can serve a tool and drive
// it from a client over an in-memory transport: the G0 transport
// round-trip. Harness-specific transports (stdio, Streamable HTTP
// against real harnesses) are Phase 1/6 concerns, not G0.
func TestMCPRoundTrip(t *testing.T) {
	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "spike", Version: "0.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo"},
		func(_ context.Context, _ *mcp.CallToolRequest, args struct {
			Text string `json:"text"`
		},
		) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "pong:" + args.Text}},
			}, nil, nil
		})

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "spike-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"text": "ping"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	if !strings.Contains(text.Text, "pong:ping") {
		t.Fatalf("unexpected tool result %q", text.Text)
	}
}
