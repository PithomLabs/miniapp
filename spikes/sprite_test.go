package spikes

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	sprites "github.com/superfly/sprites-go"
)

// TestSpriteRoundTrip proves the pinned Sprite API performs the full
// lifecycle the framework needs: provision, restrictive policy
// application with read-back verification, command execution, and
// teardown. Policy-before-exec is asserted here at the primitive
// level; the framework barrier (A3) builds on it in Phase 5.
func TestSpriteRoundTrip(t *testing.T) {
	token := os.Getenv("SPRITE_TOKEN")
	if token == "" {
		t.Skip("SPRITE_TOKEN unset (Q4): Sprite spike needs a Sprite org token")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := sprites.New(token)
	defer client.Close()

	name := fmt.Sprintf("spike-%d", time.Now().UnixNano())
	sprite, err := client.CreateSprite(ctx, name, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = client.DeleteSprite(ctx, name)
	})

	// Default-deny: the only egress a spike task needs is none.
	policy := &sprites.NetworkPolicy{Rules: []sprites.NetworkPolicyRule{
		{Action: "deny", Include: "all"},
	}}
	if err := sprite.UpdateNetworkPolicy(ctx, policy); err != nil {
		t.Fatalf("apply policy: %v", err)
	}
	readBack, err := sprite.GetNetworkPolicy(ctx)
	if err != nil {
		t.Fatalf("read back policy: %v", err)
	}
	if len(readBack.Rules) != 1 || readBack.Rules[0].Action != "deny" {
		t.Fatalf("policy read-back mismatch: %+v", readBack)
	}

	out, err := sprite.Command("echo", "spike-ok").Output()
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if string(out) != "spike-ok\n" {
		t.Fatalf("unexpected exec output %q", out)
	}
}
