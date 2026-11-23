package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"
)

// CredentialEnv carries the default credential when --credential is
// absent, so secrets never appear in process arguments.
const CredentialEnv = "MINIAPP_CREDENTIAL"

// Runner executes CLI commands against one canonical service.
type Runner struct {
	Svc    *runtime.Service
	Stdout io.Writer
	Stderr io.Writer
}

func (r *Runner) out() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return os.Stdout
}

func (r *Runner) errOut() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return os.Stderr
}

// Run executes one command line, returning the process exit code:
// 0 on success, 2 on flag misuse, 1 on service failure.
func (r *Runner) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut(), "usage: miniapp <invoke|status|signal|cancel|decide> [flags]")
		return 2
	}
	var err error
	switch args[0] {
	case "invoke":
		err = r.invoke(ctx, args[1:])
	case "status":
		err = r.status(ctx, args[1:])
	case "signal":
		err = r.signal(ctx, args[1:])
	case "cancel":
		err = r.cancel(ctx, args[1:])
	case "decide":
		err = r.decide(ctx, args[1:])
	default:
		fmt.Fprintf(r.errOut(), "unknown command %q\n", args[0])
		return 2
	}
	if err != nil {
		fmt.Fprintln(r.errOut(), "miniapp:", err)
		return 1
	}
	return 0
}

func (r *Runner) credential(fs *flag.FlagSet, args []string) (string, bool) {
	cred := fs.String("credential", "", "API credential (or "+CredentialEnv+")")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(r.errOut(), err)
		return "", false
	}
	if *cred == "" {
		*cred = os.Getenv(CredentialEnv)
	}
	return *cred, true
}

func (r *Runner) print(v any) {
	raw, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(r.out(), string(raw))
}

func (r *Runner) invoke(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("invoke", flag.ContinueOnError)
	fs.SetOutput(r.errOut())
	miniapp := fs.String("miniapp", "", "miniapp ID (required)")
	version := fs.String("version", "", "miniapp version (required)")
	tool := fs.String("tool", "", "tool name (required)")
	input := fs.String("input", "{}", "JSON input object")
	key := fs.String("key", "", "idempotency key (generated when empty)")
	trust := fs.String("trust", "", "requested trust mode")
	cred, ok := r.credential(fs, args)
	if !ok {
		return fmt.Errorf("flag error")
	}
	if *miniapp == "" || *version == "" || *tool == "" {
		return fmt.Errorf("miniapp, version, and tool are required")
	}
	out, err := r.Svc.Invoke(ctx, runtime.InvokeRequest{
		MiniappID: *miniapp, Version: *version, Tool: *tool,
		Input: json.RawMessage(*input), IdempotencyKey: *key,
		Credential: cred, RequestedTrust: contract.TrustMode(*trust),
	})
	if err != nil {
		return err
	}
	r.print(out)
	return nil
}

func (r *Runner) status(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(r.errOut())
	id := fs.String("id", "", "invocation ID (required)")
	cred, ok := r.credential(fs, args)
	if !ok {
		return fmt.Errorf("flag error")
	}
	if *id == "" {
		return fmt.Errorf("id is required")
	}
	out, err := r.Svc.Status(ctx, cred, *id)
	if err != nil {
		return err
	}
	r.print(out)
	return nil
}

func (r *Runner) signal(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("signal", flag.ContinueOnError)
	fs.SetOutput(r.errOut())
	run := fs.String("run", "", "run ID (required)")
	typ := fs.String("type", "", "signal type (required)")
	payload := fs.String("payload", "", "JSON payload")
	cred, ok := r.credential(fs, args)
	if !ok {
		return fmt.Errorf("flag error")
	}
	if *run == "" || *typ == "" {
		return fmt.Errorf("run and type are required")
	}
	sig := contract.Signal{Type: *typ, Payload: json.RawMessage(*payload)}
	if err := r.Svc.Signal(ctx, cred, *run, sig); err != nil {
		return err
	}
	r.print(map[string]string{"status": "delivered"})
	return nil
}

func (r *Runner) cancel(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(r.errOut())
	run := fs.String("run", "", "run ID (required)")
	reason := fs.String("reason", "", "cancellation reason")
	cred, ok := r.credential(fs, args)
	if !ok {
		return fmt.Errorf("flag error")
	}
	if *run == "" {
		return fmt.Errorf("run is required")
	}
	res, err := r.Svc.Cancel(ctx, cred, *run, *reason)
	if err != nil {
		return err
	}
	r.print(res)
	return nil
}

func (r *Runner) decide(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("decide", flag.ContinueOnError)
	fs.SetOutput(r.errOut())
	id := fs.String("id", "", "approval ID (required)")
	decision := fs.String("decision", "", "approve or reject (required)")
	hash := fs.String("hash", "", "intent hash digest (required)")
	cred, ok := r.credential(fs, args)
	if !ok {
		return fmt.Errorf("flag error")
	}
	if *id == "" || *decision == "" || *hash == "" {
		return fmt.Errorf("id, decision, and hash are required")
	}
	if err := r.Svc.DecideApproval(ctx, cred, *id, contract.ApprovalDecision(*decision), contract.Digest(*hash)); err != nil {
		return err
	}
	r.print(map[string]string{"status": "decided"})
	return nil
}
