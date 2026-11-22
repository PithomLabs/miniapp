package store

import (
	"strings"
	"testing"
)

// canary is a unique credential stand-in. No test output may ever
// contain it; its presence proves a redaction failure.
const canary = "canary-9f2c-redact-me"

func TestRedactDSNTable(t *testing.T) {
	cases := []struct {
		name     string
		dsn      string
		mustHold []string
	}{
		{"url-with-password",
			"postgres://u:" + canary + "@host:5432/db",
			[]string{"u@host", "host:5432"}},
		{"url-without-password",
			"postgres://u@host:5432/db",
			[]string{"u@host"}},
		{"url-query-password",
			"postgres://host/db?password=" + canary,
			[]string{"host"}},
		{"keyword-unquoted",
			"host=db user=u password=" + canary + " dbname=x",
			[]string{"host=db", "user=u"}},
		{"keyword-quoted",
			"host=db password='" + canary + " secret' dbname=x",
			[]string{"host=db", "dbname=x"}},
		{"keyword-escaped-quote",
			`host=db password='a\'` + canary + `' dbname=x`,
			[]string{"host=db"}},
		{"keyword-passfile",
			"host=db passfile=/tmp/" + canary + " dbname=x",
			[]string{"host=db"}},
		{"keyword-uppercase-key",
			"host=db PASSWORD=" + canary,
			[]string{"host=db"}},
		{"malformed-unterminated",
			"host=db password='oops",
			[]string{"[redacted:unparseable-dsn]"}},
		{"malformed-url",
			"postgres://[::1",
			[]string{"[redacted:unparseable-dsn]"}},
		{"malformed-no-key",
			"=orphan host=db",
			[]string{"[redacted:unparseable-dsn]"}},
		{"bare-dbname",
			"mydb",
			[]string{"mydb"}},
		{"empty",
			"",
			[]string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := PoolConfig{DSN: c.dsn, Mode: PoolDirect, MaxConns: 1}.String()
			if strings.Contains(out, canary) {
				t.Fatalf("credential leaked: %s", out)
			}
			for _, want := range c.mustHold {
				if !strings.Contains(out, want) {
					t.Errorf("missing %q in: %s", want, out)
				}
			}
		})
	}
}

func TestRedactDoubledQuoteFailsClosed(t *testing.T) {
	// Doubled single-quotes are not conninfo escapes; the tokenizer
	// must reject the shape rather than emit a half-parsed password.
	out := PoolConfig{DSN: "host=db password='it''" + canary + "' dbname=x"}.String()
	if strings.Contains(out, canary) {
		t.Fatalf("credential leaked: %s", out)
	}
	if !strings.Contains(out, "[redacted:unparseable-dsn]") {
		t.Fatalf("expected full redaction, got: %s", out)
	}
}
