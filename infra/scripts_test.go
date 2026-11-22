package infra

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var repoScripts = []string{
	"scripts/dev/postgres-image.sh",
	"scripts/dev/postgres-volume-check.sh",
	"scripts/dev/local-secrets-init.sh",
	"scripts/fly/secret-input.sh",
	"scripts/fly/app-preflight.sh",
	"scripts/fly/secret-apply.sh",
	"scripts/fly/release-verify.sh",
	"scripts/checks.sh",
}

func bashPath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	return p
}

func TestScriptsSyntaxValid(t *testing.T) {
	bash := bashPath(t)
	for _, s := range repoScripts {
		cmd := exec.Command(bash, "-n", filepath.Join(repoRoot(t), s))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s syntax invalid: %v\n%s", s, err, out)
		}
	}
}

func TestScriptsTracingDisabled(t *testing.T) {
	for _, s := range repoScripts {
		content := readRepoFile(t, s)
		if !strings.Contains(content, "set -u") && !strings.Contains(content, "set -eu") {
			t.Errorf("%s must fail closed with set -u", s)
		}
		for _, line := range strings.Split(content, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "#") {
				continue
			}
			if strings.Contains(trim, "set -x") {
				t.Errorf("%s enables tracing: %q", s, trim)
			}
		}
	}
}

func TestSecretInputHandling(t *testing.T) {
	content := readRepoFile(t, "scripts/fly/secret-input.sh")
	for _, want := range []string{"read -rs", "secrets import", "unset value", "--stage", "miniapp-demo", "MINIAPP_PROD_APP"} {
		if !strings.Contains(content, want) {
			t.Errorf("secret-input.sh missing %q", want)
		}
	}
	for _, line := range strings.Split(content, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			continue
		}
		// Only echo arguments matter: a static message mentioning the
		// word "value" is safe; expanding $value into echo is not.
		parts := strings.Split(trim, "echo")
		for _, arg := range parts[1:] {
			if strings.Contains(arg, "$value") || strings.Contains(arg, "${value}") {
				t.Errorf("secret-input.sh may print the value: %q", trim)
			}
		}
	}
	apply := readRepoFile(t, "scripts/fly/secret-apply.sh")
	if !strings.Contains(apply, "secrets list") || !strings.Contains(apply, "secrets deploy") {
		t.Error("secret-apply.sh must list metadata, confirm, then deploy staged secrets")
	}
}

func TestVolumeCheckNeverDeletes(t *testing.T) {
	content := readRepoFile(t, "scripts/dev/postgres-volume-check.sh")
	if strings.Contains(content, "volume rm") || strings.Contains(content, "volume prune") {
		t.Error("volume check must never delete volumes")
	}
	if !strings.Contains(content, "MINIAPP_ALLOW_FRESH_DB") {
		t.Error("volume check must offer an explicit fresh-start override")
	}
}

// TestVolumeCheckBehavior exercises the refusal logic with a fake
// docker that only answers `volume ls`. No daemon required.
func TestVolumeCheckBehavior(t *testing.T) {
	bash := bashPath(t)
	shimDir := t.TempDir()
	shim := "#!/usr/bin/env bash\nif [ \"$1 $2\" = \"volume ls\" ]; then printf '%s\\n' $SHIM_VOLUMES; fi\n"
	shimPath := filepath.Join(shimDir, "docker")
	if err := os.WriteFile(shimPath, []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repoRoot(t), "scripts/dev/postgres-volume-check.sh")
	run := func(volumes, allow string) (int, string) {
		cmd := exec.Command(bash, script, "19", "miniapp-pgdata")
		cmd.Env = append(os.Environ(),
			"PATH="+shimDir+":"+os.Getenv("PATH"),
			"SHIM_VOLUMES="+volumes,
			"MINIAPP_ALLOW_FRESH_DB="+allow,
		)
		out, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		return code, string(out)
	}
	if code, _ := run("", ""); code != 0 {
		t.Error("fresh start with no volumes must succeed")
	}
	if code, _ := run("miniapp-pgdata-19", ""); code != 0 {
		t.Error("matching volume must succeed")
	}
	if code, out := run("miniapp-pgdata-18", ""); code == 0 {
		t.Error("incompatible existing volume must refuse startup")
	} else if !strings.Contains(out, "REFUSED") {
		t.Errorf("refusal must explain itself: %s", out)
	}
	if code, _ := run("miniapp-pgdata-18", "1"); code != 0 {
		t.Error("explicit fresh-start override must succeed")
	}
}

// TestLocalSecretsInit exercises real file behavior in a temp dir:
// owner-only permissions, generated credentials, idempotency, and no
// overwrite of existing secrets. Captured output must not leak values.
func TestLocalSecretsInit(t *testing.T) {
	bash := bashPath(t)
	dir := t.TempDir()
	secretDir := filepath.Join(dir, "miniapp")
	script := filepath.Join(repoRoot(t), "scripts/dev/local-secrets-init.sh")
	run := func() string {
		cmd := exec.Command(bash, script, secretDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("secrets init failed: %v\n%s", err, out)
		}
		return string(out)
	}
	out1 := run()
	data, err := os.ReadFile(filepath.Join(secretDir, "db_password"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 32 {
		t.Fatalf("generated credential too short: %d bytes", len(data))
	}
	if strings.Contains(out1, string(data)) {
		t.Fatal("script output leaks the credential value")
	}
	st, err := os.Stat(secretDir)
	if err != nil || st.Mode().Perm() != 0o700 {
		t.Fatalf("secrets dir perms = %v, want 0700", st.Mode())
	}
	fst, err := os.Stat(filepath.Join(secretDir, "db_password"))
	if err != nil || fst.Mode().Perm() != 0o600 {
		t.Fatalf("secret file perms = %v, want 0600", fst.Mode())
	}
	run()
	data2, err := os.ReadFile(filepath.Join(secretDir, "db_password"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(data2) {
		t.Fatal("second run overwrote the existing credential")
	}
}

func TestGitignoreExcludesSecrets(t *testing.T) {
	content := readRepoFile(t, ".gitignore")
	for _, want := range []string{".env", ".pem", ".key", "secrets/"} {
		if !strings.Contains(content, want) {
			t.Errorf(".gitignore missing %q", want)
		}
	}
}
