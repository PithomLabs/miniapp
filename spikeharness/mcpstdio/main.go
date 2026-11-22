// Command mcp-spike is a test-only MCP server over stdio for the G0-Core
// condition-6 harness round trip. It exposes one tool, spike_echo, that
// validates a presenter test credential before responding.
//
// This is spike scaffolding, not framework code: production auth,
// routing, and tools arrive in Phase 1/2. It binds no network ports,
// touches no database, and holds no real credentials: MCP_SPIKE_TOKEN
// must carry a disposable test value (fail-closed when unset), and
// neither the token nor any transport detail is ever logged.
package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type echoArgs struct {
	AuthToken string `json:"auth_token"`
	Text      string `json:"text"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mcp-spike:", err)
		os.Exit(1)
	}
}

func run() error {
	expected := os.Getenv("MCP_SPIKE_TOKEN")
	if expected == "" {
		return fmt.Errorf("MCP_SPIKE_TOKEN is required (disposable test value)")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := mcp.NewServer(&mcp.Implementation{Name: "miniapp-spike", Version: "0.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "spike_echo",
		Description: "Test-only echo. Requires auth_token matching the server's test credential.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args echoArgs) (*mcp.CallToolResult, any, error) {
		if subtle.ConstantTimeCompare([]byte(args.AuthToken), []byte(expected)) != 1 {
			fmt.Fprintln(os.Stderr, "mcp-spike: spike_echo DENIED")
			return nil, nil, fmt.Errorf("invalid auth token")
		}
		fmt.Fprintln(os.Stderr, "mcp-spike: spike_echo OK")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "spike-ok:" + args.Text}},
		}, nil, nil
	})
	return server.Run(ctx, &mcp.StdioTransport{})
}
