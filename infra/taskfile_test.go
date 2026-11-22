package infra

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(dir)
}

func taskPath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("task")
	if err != nil {
		t.Skip("task binary not installed")
	}
	return p
}

func listTasks(t *testing.T, taskfile string) string {
	t.Helper()
	cmd := exec.Command(taskPath(t), "--taskfile", taskfile, "--list-all", "--silent")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s --list-all failed: %v\n%s", taskfile, err, out)
	}
	return string(out)
}

var devTasks = []string{
	"doctor", "dev:db:pull", "dev:db:up", "dev:db:down", "dev:db:logs",
	"dev:db:status", "dev:db:reset", "dev:up", "dev:down", "test",
	"test:integration", "fly:status", "fly:secrets:list", "fly:secrets:stage",
	"fly:secrets:apply", "fly:deploy",
}

var prodTasks = []string{
	"doctor", "fly:status", "secrets:list", "secrets:stage",
	"secrets:apply", "deploy", "release:verify", "rollback:prepare",
}

func TestDevTaskInventory(t *testing.T) {
	out := listTasks(t, "Taskfile.dev.yml")
	for _, name := range devTasks {
		if !strings.Contains(out, name) {
			t.Errorf("dev task %q missing from --list-all output", name)
		}
	}
}

func TestProdTaskInventory(t *testing.T) {
	out := listTasks(t, "Taskfile.prod.yml")
	for _, name := range prodTasks {
		if !strings.Contains(out, name) {
			t.Errorf("prod task %q missing from --list-all output", name)
		}
	}
}

func readRepoFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestTaskfilesIndependent(t *testing.T) {
	dev := readRepoFile(t, "Taskfile.dev.yml")
	prod := readRepoFile(t, "Taskfile.prod.yml")
	for _, content := range []string{dev, prod} {
		if strings.Contains(content, "includes:") {
			t.Error("Taskfiles must not import each other (found includes:)")
		}
	}
	if strings.Contains(dev, "MINIAPP_PROD_APP") {
		t.Error("dev Taskfile must not reference production configuration")
	}
	for _, s := range []string{"miniapp-demo", "compose.yaml", "dev:db", "POSTGRES_IMAGE"} {
		if strings.Contains(prod, s) {
			t.Errorf("prod Taskfile must not reference development surface %q", s)
		}
	}
}

func TestDestructiveOpsGated(t *testing.T) {
	dev := readRepoFile(t, "Taskfile.dev.yml")
	if !strings.Contains(dev, "prompt:") {
		t.Error("dev:db:reset must carry an explicit confirmation prompt")
	}
	prod := readRepoFile(t, "Taskfile.prod.yml")
	if !strings.Contains(prod, "Type the production") {
		t.Error("prod deploy must require typed application confirmation")
	}
	// Neither inspection task may deploy.
	lines := strings.Split(prod, "\n")
	inVerify := false
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasSuffix(trim, ":") && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") {
			inVerify = false
		}
		if trim == "release:verify:" || trim == "rollback:prepare:" {
			inVerify = true
		}
		if inVerify && strings.Contains(l, "flyctl deploy") && !strings.HasPrefix(trim, "- echo") {
			t.Errorf("inspection task must not deploy: %q", trim)
		}
	}
}

func TestNoSecretInterpolationInTasks(t *testing.T) {
	for _, f := range []string{"Taskfile.dev.yml", "Taskfile.prod.yml"} {
		content := readRepoFile(t, f)
		if strings.Contains(content, "POSTGRES_PASSWORD=") {
			t.Errorf("%s interpolates a credential literal", f)
		}
	}
	dev := readRepoFile(t, "Taskfile.dev.yml")
	// The integration task must clear its transient password variable.
	if strings.Contains(dev, "DBPW=") && !strings.Contains(dev, "unset DBPW") {
		t.Error("test:integration must unset its transient password variable")
	}
}

// TestGofmtGatesFail requires every gofmt gate to fail the run on
// unformatted files: bare `gofmt -l` always exits 0 and proves
// nothing. Each gate must capture the listing, propagate a genuine
// gofmt execution error, and fail on non-empty output.
func TestGofmtGatesFail(t *testing.T) {
	dev := readRepoFile(t, "Taskfile.dev.yml")
	testCmds := taskCmds(t, dev, "test")
	found := false
	for _, c := range testCmds {
		if !strings.Contains(c, "gofmt -l") {
			continue
		}
		found = true
		if !strings.Contains(c, "test -z") {
			t.Errorf("dev test gate does not fail on output: %q", c)
		}
		if !strings.Contains(c, "rc=") && !strings.Contains(c, "$?") {
			t.Errorf("dev test gate ignores gofmt execution errors: %q", c)
		}
	}
	if !found {
		t.Error("dev test task has no gofmt gate")
	}

	ci := readRepoFile(t, ".github/workflows/ci.yml")
	ciFound := false
	for _, l := range strings.Split(ci, "\n") {
		if !strings.Contains(l, "gofmt -l") {
			continue
		}
		ciFound = true
		if !strings.Contains(l, "test -z") {
			t.Errorf("CI gofmt gate does not fail on output: %q", strings.TrimSpace(l))
		}
		if !strings.Contains(l, "rc=") && !strings.Contains(l, "$?") {
			t.Errorf("CI gofmt gate ignores gofmt execution errors: %q", strings.TrimSpace(l))
		}
	}
	if !ciFound {
		t.Error("CI workflow has no gofmt gate")
	}

	checks := readRepoFile(t, "scripts/checks.sh")
	if !strings.Contains(checks, "gofmt -l .") {
		t.Error("checks.sh must cover the whole tree (including future packages)")
	}
	if strings.Contains(checks, "gofmt -l contract") {
		t.Error("checks.sh still pins a package subset for gofmt")
	}
	if !strings.Contains(checks, "fmt_rc") {
		t.Error("checks.sh must propagate gofmt execution errors")
	}
}

// taskCmds returns the command lines of one task from a Taskfile body.
// Task headers are two-space-indented "name:" lines; task bodies run
// until the next such header.
func taskCmds(t *testing.T, body, name string) []string {
	t.Helper()
	var cmds []string
	inTask := false
	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "  ") && !strings.HasPrefix(l, "   ") &&
			strings.HasSuffix(strings.TrimSpace(l), ":") {
			inTask = strings.TrimSpace(l) == name+":"
			continue
		}
		if inTask && strings.HasPrefix(strings.TrimSpace(l), "- ") {
			cmds = append(cmds, strings.TrimSpace(l)[2:])
		}
	}
	if len(cmds) == 0 {
		t.Fatalf("task %q not found or has no commands", name)
	}
	return cmds
}

// TestReadinessGatesUnmasked requires every retry loop over pg_isready
// to be followed, within the same task, by a standalone final gate
// whose failure cannot be masked by a trailing sleep, echo, or
// fallback. A loop whose last iteration ends in `|| sleep` exits 0
// after total failure; only the final gate makes the task honest.
func TestReadinessGatesUnmasked(t *testing.T) {
	dev := readRepoFile(t, "Taskfile.dev.yml")
	for _, task := range []string{"dev:db:up", "test:integration"} {
		cmds := taskCmds(t, dev, task)
		loopAt := -1
		for i, c := range cmds {
			if strings.Contains(c, "for i in") && strings.Contains(c, "pg_isready") {
				loopAt = i
			}
		}
		if loopAt < 0 {
			t.Errorf("%s: expected retry loop not found", task)
			continue
		}
		gated := false
		for _, c := range cmds[loopAt+1:] {
			if !strings.Contains(c, "pg_isready") {
				continue
			}
			if strings.Contains(c, "for i in") {
				continue
			}
			if strings.Contains(c, "|| sleep") || strings.Contains(c, "|| true") || strings.Contains(c, "|| echo") {
				t.Errorf("%s: final gate masks failure: %q", task, c)
				continue
			}
			gated = true
		}
		if !gated {
			t.Errorf("%s: retry loop has no unmasked final readiness gate after it", task)
		}
	}
}
