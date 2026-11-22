package infra

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func dockerPath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker CLI not installed")
	}
	return p
}

// TestComposeValidates parses the Compose files without contacting a
// daemon. Daemon-backed behavior is verified live, never inferred here.
func TestComposeValidates(t *testing.T) {
	docker := dockerPath(t)
	dir := repoRoot(t)
	env := append(os.Environ(), "MINIAPP_SECRETS_DIR="+t.TempDir())
	for _, args := range [][]string{
		{"compose", "-f", "compose.yaml", "config", "--quiet"},
		{"compose", "-f", "compose.yaml", "-f", "compose.host.yaml", "config", "--quiet"},
	} {
		cmd := exec.Command(docker, args...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("compose %v failed: %v\n%s", args, err, out)
		}
	}
}

func composeConfig(t *testing.T, files ...string) string {
	t.Helper()
	docker := dockerPath(t)
	args := []string{"compose"}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "config")
	cmd := exec.Command(docker, args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "MINIAPP_SECRETS_DIR="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compose config failed: %v\n%s", err, out)
	}
	return string(out)
}

func TestComposeBaseHasNoPublishedPorts(t *testing.T) {
	out := composeConfig(t, "compose.yaml")
	if strings.Contains(out, "ports:") {
		t.Error("base compose.yaml must not publish ports")
	}
	if !strings.Contains(out, "image: postgres:19.0") {
		t.Error("base compose must use the resolved postgres:19.0 default")
	}
	if !strings.Contains(out, "miniapp-pgdata-19") {
		t.Error("base compose must use the major-isolated volume name")
	}
	if !strings.Contains(out, "POSTGRES_PASSWORD_FILE") {
		t.Error("base compose must use file-based password delivery")
	}
}

func TestComposeHostAccessIsLoopbackOnly(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "compose.host.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var code strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if trimmed := strings.TrimSpace(line); !strings.HasPrefix(trimmed, "#") {
			code.WriteString(line + "\n")
		}
	}
	if !strings.Contains(code.String(), "127.0.0.1") {
		t.Error("host override must bind loopback explicitly")
	}
	if strings.Contains(code.String(), "0.0.0.0") {
		t.Error("host override must never bind 0.0.0.0")
	}
}

func TestComposeHasNoCredentialLiterals(t *testing.T) {
	for _, f := range []string{"compose.yaml", "compose.host.yaml"} {
		content := readRepoFile(t, f)
		if strings.Contains(content, "POSTGRES_PASSWORD:") {
			t.Errorf("%s sets a literal password", f)
		}
	}
}
