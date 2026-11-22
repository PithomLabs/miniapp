package store

import (
	"strings"
	"testing"
)

func TestPoolConfigRedactsPassword(t *testing.T) {
	c := PoolConfig{DSN: "postgres://user:s3cret@host:5432/db", Mode: PoolDirect, MaxConns: 4}
	s := c.String()
	if strings.Contains(s, "s3cret") {
		t.Fatalf("password leaked: %s", s)
	}
	if !strings.Contains(s, "user@host") {
		t.Fatalf("username/host lost: %s", s)
	}
}

func TestTenantContextSQL(t *testing.T) {
	q := TenantContextSQL()
	if !strings.Contains(q, "app.tenant_id") || !strings.Contains(q, "$1") {
		t.Fatalf("unexpected tenant context SQL: %s", q)
	}
}
