package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// VerifyRevocation implements `miniapp auth verify-revocation
// <expected-hash>`: it polls every listed replica's /healthz and
// reports whether the fleet holds the expected grant
// configuration. Inventory comes only from explicit operator
// input (--replicas and/or --replicas-file); there is no
// discovery. Exit code: 0 iff every listed replica matches,
// 2 on flag misuse or empty inventory, 1 otherwise. It never
// reports success on an incomplete inventory.
//
// Per-replica verdicts: ok (hash equals expected), stale
// (reachable but different — the replica needs a restart),
// unreachable (poll/timeout/parse failure). A mixed fleet is
// summarized as MISMATCHED (partial rollout or drift).
func VerifyRevocation(ctx context.Context, stdout, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("verify-revocation", flag.ContinueOnError)
	fs.SetOutput(stderr)
	replicas := fs.String("replicas", "", "comma-separated replica bases (host[:port] or http(s):// URL)")
	replicasFile := fs.String("replicas-file", "", "file with one replica base per line (# comments allowed)")
	timeout := fs.String("timeout", "5s", "per-replica poll timeout (Go duration)")
	// The hash may lead (documented shape) or trail the flags;
	// Go's flag parser stops at the first positional, so split
	// a leading hash off explicitly.
	restArgs := args
	expected := ""
	if len(restArgs) > 0 && !strings.HasPrefix(restArgs[0], "-") {
		expected = restArgs[0]
		restArgs = restArgs[1:]
	}
	if err := fs.Parse(restArgs); err != nil {
		return 2
	}
	if rest := fs.Args(); len(rest) == 1 && expected == "" {
		expected = rest[0]
	} else if len(rest) != 0 {
		fmt.Fprintln(stderr, "usage: miniapp auth verify-revocation <expected-hash> [--replicas h,...] [--replicas-file f]")
		return 2
	}
	if expected == "" {
		fmt.Fprintln(stderr, "usage: miniapp auth verify-revocation <expected-hash> [--replicas h,...] [--replicas-file f]")
		return 2
	}
	dur, err := time.ParseDuration(*timeout)
	if err != nil || dur <= 0 {
		fmt.Fprintln(stderr, "miniapp: invalid --timeout (want positive Go duration)")
		return 2
	}
	targets, err := revocationInventory(*replicas, *replicasFile)
	if err != nil {
		fmt.Fprintln(stderr, "miniapp:", err)
		return 2
	}
	if len(targets) == 0 {
		fmt.Fprintln(stderr, "miniapp: no replicas listed (refusing vacuous success)")
		return 2
	}
	client := &http.Client{Timeout: dur}
	matched := 0
	verdicts := make([]string, 0, len(targets))
	for _, tgt := range targets {
		v, detail := pollHealthz(ctx, client, tgt, expected)
		verdicts = append(verdicts, v)
		if v == "ok" {
			matched++
		}
		if detail != "" {
			fmt.Fprintf(stdout, "%s: %s (%s)\n", tgt, v, detail)
		} else {
			fmt.Fprintf(stdout, "%s: %s\n", tgt, v)
		}
	}
	fmt.Fprintf(stdout, "match: %d/%d\n", matched, len(targets))
	switch {
	case matched == len(targets):
		fmt.Fprintln(stdout, "revocation verified: every listed replica holds the expected configuration")
		return 0
	case matched == 0 && !contains(verdicts, "unreachable"):
		fmt.Fprintln(stdout, "STALE: no listed replica holds the expected configuration (restart replicas)")
		return 1
	case !contains(verdicts, "ok"):
		fmt.Fprintln(stdout, "INCOMPLETE: no listed replica confirmed (unreachable; inventory unverified)")
		return 1
	default:
		fmt.Fprintln(stdout, "MISMATCHED: the fleet disagrees (partial rollout, drift, or outage)")
		return 1
	}
}

func contains(vs []string, want string) bool {
	for _, v := range vs {
		if v == want {
			return true
		}
	}
	return false
}

// revocationInventory merges --replicas and --replicas-file into
// an ordered, deduplicated target list. Bare host[:port]
// entries resolve to http://host[:port] (default port 8080,
// the established REST default); http(s):// URLs pass through.
func revocationInventory(csv, file string) ([]string, error) {
	var raw []string
	if csv != "" {
		raw = append(raw, strings.Split(csv, ",")...)
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("replicas file: %w", err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if i := strings.Index(line, "#"); i >= 0 {
				line = line[:i]
			}
			if strings.TrimSpace(line) != "" {
				raw = append(raw, line)
			}
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, r := range raw {
		r = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(r), "/"))
		if r == "" {
			continue
		}
		if !strings.Contains(r, "://") {
			if !strings.Contains(r, ":") {
				r += ":8080"
			}
			r = "http://" + r
		}
		if seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out, nil
}

func pollHealthz(ctx context.Context, client *http.Client, base, expected string) (verdict, detail string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/healthz", nil)
	if err != nil {
		return "unreachable", "bad target"
	}
	resp, err := client.Do(req)
	if err != nil {
		return "unreachable", "poll failed"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "unreachable", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	var body struct {
		GrantConfigHash string `json:"grant_config_hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.GrantConfigHash == "" {
		return "unreachable", "unparseable health body"
	}
	// String comparison only: consumers never re-canonicalize;
	// the hash is authoritative from auth.Policy on each replica.
	if body.GrantConfigHash != expected {
		return "stale", "holds " + body.GrantConfigHash
	}
	return "ok", ""
}
