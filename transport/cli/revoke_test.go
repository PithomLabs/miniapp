package cli_test

// Prompt28 workstream D — verify-revocation CLI matrix. Pure
// unit tests over httptest health endpoints (no database):
// exit codes, per-replica verdicts, inventory rules, and the
// never-success-on-incomplete guarantee.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp/transport/cli"
)

func p28HealthServer(t *testing.T, hash string, delay time.Duration, code int, body string) *httptest.Server {
	t.Helper()
	if body == "" {
		body = fmt.Sprintf(`{"status":"ok","grant_config_hash":%q}`, hash)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(code)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func p28RunVerify(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	code := cli.VerifyRevocation(ctx, &stdout, &stderr, args)
	if code == 2 && stderr.Len() == 0 {
		t.Errorf("usage error without usage text (args %v)", args)
	}
	return code, stdout.String()
}

func TestP28VerifyRevocationMatrix(t *testing.T) {
	const want = "sha256:expected"
	ok1 := p28HealthServer(t, want, 0, 200, "")
	ok2 := p28HealthServer(t, want, 0, 200, "")
	stale := p28HealthServer(t, "sha256:old", 0, 200, "")
	broken := p28HealthServer(t, "", 0, 200, "not-json{{{")
	slow := p28HealthServer(t, want, 1500*time.Millisecond, 200, "")

	cases := []struct {
		name string
		args []string
		code int
		want []string
	}{
		{"all-match", []string{want, "--replicas", ok1.URL + "," + ok2.URL}, 0,
			[]string{"ok", "ok", "match: 2/2", "revocation verified"}},
		{"one-stale", []string{want, "--replicas", ok1.URL + "," + stale.URL}, 1,
			[]string{"ok", "stale", "match: 1/2", "MISMATCHED"}},
		{"all-stale", []string{want, "--replicas", stale.URL}, 1,
			[]string{"stale", "match: 0/1", "STALE"}},
		{"unreachable", []string{want, "--replicas", "http://127.0.0.1:9"}, 1,
			[]string{"unreachable", "INCOMPLETE"}},
		{"unparseable", []string{want, "--replicas", broken.URL}, 1,
			[]string{"unreachable"}},
		{"timeout", []string{want, "--timeout", "50ms", "--replicas", slow.URL}, 1,
			[]string{"unreachable"}},
		{"empty-inventory", []string{want}, 2, nil},
		{"missing-hash", []string{"--replicas", ok1.URL}, 2, nil},
		{"bad-timeout", []string{want, "--timeout", "soon", "--replicas", ok1.URL}, 2, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out := p28RunVerify(t, tc.args...)
			if code != tc.code {
				t.Errorf("exit = %d, want %d (out %q)", code, tc.code, out)
			}
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("output %q lacks %q", out, w)
				}
			}
		})
	}
}

func TestP28VerifyRevocationFileInventory(t *testing.T) {
	const want = "sha256:expected"
	ok := p28HealthServer(t, want, 0, 200, "")
	file := filepath.Join(t.TempDir(), "replicas.txt")
	content := "# fleet inventory\n\n" + ok.URL + "\n" + ok.URL + "  \n# trailing comment\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	code, out := p28RunVerify(t, want, "--replicas-file", file)
	if code != 0 {
		t.Errorf("exit = %d, want 0 (out %q)", code, out)
	}
	if got := strings.Count(out, "ok"); got != 1 {
		t.Errorf("ok lines = %d, want 1 (deduped inventory)", got)
	}
	if code, _ := p28RunVerify(t, want, "--replicas-file", filepath.Join(t.TempDir(), "missing.txt")); code != 2 {
		t.Errorf("missing file exit = %d, want 2", code)
	}
}
