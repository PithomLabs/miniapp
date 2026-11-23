// Command miniapp serves the Phase 1 framework: REST and MCP
// adapters over the canonical invocation service, direct operator
// verbs, and migrations.
//
// Configuration (environment, *_FILE variants supported where noted
// by internal/config):
//
//	DATABASE_URL            PostgreSQL DSN (or DATABASE_URL_FILE)
//	MINIAPP_KEYS_FILE       JSON credential bindings (required)
//	MINIAPP_GRANTS_FILE     JSON tenant grants (required)
//	MINIAPP_REST_PORT       REST listen port (serve-rest; default 8080)
//	MINIAPP_DEMO_PROBE      "1" registers the test-only Class D probe
//	                        (demo use only; never production)
//
// The keys/grants files are demo scaffolding: disposable,
// owner-generated JSON outside the repository. Production
// authentication policy arrives with a later phase.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor/classa"
	dbosexec "github.com/PithomLabs/miniapp/executor/dbos"
	"github.com/PithomLabs/miniapp/internal/config"
	crmread "github.com/PithomLabs/miniapp/miniapps/crm-read"
	classd "github.com/PithomLabs/miniapp/probe/classd"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
	"github.com/PithomLabs/miniapp/transport/cli"
	"github.com/PithomLabs/miniapp/transport/mcp"
	"github.com/PithomLabs/miniapp/transport/rest"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve-rest":
		err = serveRest()
	case "serve-mcp":
		err = serveMCP()
	case "migrate":
		err = migrateUp()
	case "invoke", "status", "signal", "cancel", "decide":
		err = runDirect(os.Args[1:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "miniapp:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: miniapp <serve-rest|serve-mcp|migrate|invoke|status|signal|cancel|decide> ...")
}

type wired struct {
	svc      *runtime.Service
	pool     *pgxpool.Pool
	executor *dbosexec.Executor
	logger   *slog.Logger
}

func (w *wired) close() {
	if w.executor != nil {
		_ = w.executor.Shutdown(30 * time.Second)
	}
	if w.pool != nil {
		w.pool.Close()
	}
}

type keyFileEntry struct {
	Agent  string   `json:"agent"`
	Kind   string   `json:"kind"`
	ID     string   `json:"id"`
	Tenant string   `json:"tenant"`
	Roles  []string `json:"roles"`
}

func loadPolicy() (auth.Policy, error) {
	keysPath := os.Getenv("MINIAPP_KEYS_FILE")
	grantsPath := os.Getenv("MINIAPP_GRANTS_FILE")
	if keysPath == "" || grantsPath == "" {
		return auth.Policy{}, fmt.Errorf("MINIAPP_KEYS_FILE and MINIAPP_GRANTS_FILE are required")
	}
	keysRaw, err := os.ReadFile(keysPath)
	if err != nil {
		return auth.Policy{}, fmt.Errorf("keys file: %w", err)
	}
	var keys map[string]keyFileEntry
	if err := json.Unmarshal(keysRaw, &keys); err != nil {
		return auth.Policy{}, fmt.Errorf("keys file: %w", err)
	}
	grantsRaw, err := os.ReadFile(grantsPath)
	if err != nil {
		return auth.Policy{}, fmt.Errorf("grants file: %w", err)
	}
	var grants map[string]map[string]contract.TrustMode
	if err := json.Unmarshal(grantsRaw, &grants); err != nil {
		return auth.Policy{}, fmt.Errorf("grants file: %w", err)
	}
	policy := auth.Policy{Keys: map[string]auth.Binding{}, Grants: grants}
	for cred, e := range keys {
		if cred == "" || e.Agent == "" || e.Tenant == "" {
			return auth.Policy{}, fmt.Errorf("keys file: credential, agent, and tenant are all required")
		}
		policy.Keys[cred] = auth.Binding{AgentID: e.Agent, Authority: contract.Authority{
			Kind: contract.AuthorityKind(e.Kind), ID: e.ID, TenantID: e.Tenant,
		}, Roles: e.Roles}
	}
	// Fail closed on unusable bindings before serving anything.
	for cred, b := range policy.Keys {
		if _, err := policy.Authenticate(cred); err != nil {
			return auth.Policy{}, fmt.Errorf("keys file: %w", err)
		}
		_ = b
	}
	return policy, nil
}

func buildService(ctx context.Context, logger *slog.Logger) (*wired, error) {
	dsn, err := config.DatabaseURL()
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	reg := registry.New()
	if err := crmread.Register(reg); err != nil {
		pool.Close()
		return nil, err
	}
	if os.Getenv("MINIAPP_DEMO_PROBE") == "1" {
		logger.Warn("demo probe registered (test-only miniapp; never production)")
		if err := classd.Register(reg); err != nil {
			pool.Close()
			return nil, err
		}
	}
	policy, err := loadPolicy()
	if err != nil {
		pool.Close()
		return nil, err
	}
	ex, err := dbosexec.New(dbosexec.Config{Pool: pool, Registry: reg, Logger: logger})
	if err != nil {
		pool.Close()
		return nil, err
	}
	eff := effects.New(effects.Config{Pool: pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(pool)},
		Canceller: runtime.EffectCanceller(pool),
		Policy:    effects.StrictPolicy{Pool: pool, Registry: reg, Grants: policy},
	})
	classd.Effects = eff
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: pool,
		Sync: classa.New(pool, reg), Async: ex,
		Approver: approvals.New(pool), Auditor: audit.New(pool, logger), Effects: eff,
	})
	if err != nil {
		_ = ex.Shutdown(10 * time.Second)
		pool.Close()
		return nil, err
	}
	return &wired{svc: svc, pool: pool, executor: ex, logger: logger}, nil
}

func serveRest() error {
	port := os.Getenv("MINIAPP_REST_PORT")
	if port == "" {
		port = "8080"
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	w, err := buildService(ctx, logger)
	if err != nil {
		return err
	}
	defer w.close()
	srv := &http.Server{Addr: "127.0.0.1:" + port, Handler: rest.New(w.svc).Handler()}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	logger.Info("rest serving", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func serveMCP() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	w, err := buildService(ctx, logger)
	if err != nil {
		return err
	}
	defer w.close()
	return mcp.NewServer(w.svc).Run(ctx, &sdk.StdioTransport{})
}

func runDirect(args []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	w, err := buildService(ctx, logger)
	if err != nil {
		return err
	}
	defer w.close()
	r := &cli.Runner{Svc: w.svc}
	if code := r.Run(ctx, args); code != 0 {
		return fmt.Errorf("command failed (exit %d)", code)
	}
	return nil
}

func migrateUp() error {
	dsn, err := config.DatabaseURL()
	if err != nil {
		return err
	}
	url := dsn
	if s, ok := cutPrefix(dsn, "postgres://"); ok {
		url = "pgx5://" + s
	} else if s, ok := cutPrefix(dsn, "postgresql://"); ok {
		url = "pgx5://" + s
	}
	m, err := migrate.New("file://migrations", url)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err.Error() != "no change" {
		return err
	}
	fmt.Println("migrations up to date")
	return nil
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}
