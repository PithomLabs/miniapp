package store

import (
	"net/url"
	"strings"
)

// redactedDSN replaces a DSN that cannot be sanitized safely. It is a
// constant: no input bytes survive redaction failure.
const redactedDSN = "[redacted:unparseable-dsn]"

// redactDSN removes credential material from a PostgreSQL DSN for
// diagnostics. It handles URL DSNs (userinfo password plus password
// and passfile query parameters) and keyword DSNs (password and
// passfile keywords, quote-aware). Anything that cannot be parsed
// safely redacts entirely; the original is never printed on failure.
func redactDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql" || u.User != nil || u.Host != "") {
		return redactURL(u)
	}
	if strings.Contains(dsn, "://") {
		return redactedDSN
	}
	if !strings.Contains(dsn, "=") {
		// Bare database name or host: no vector carries a password
		// without "=" or URL userinfo.
		return dsn
	}
	out, ok := redactKeywords(dsn)
	if !ok {
		return redactedDSN
	}
	return out
}

func redactURL(u *url.URL) string {
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	if q := u.Query(); len(q) > 0 {
		q.Del("password")
		q.Del("passfile")
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// redactKeywords sanitizes a libpq keyword DSN, preserving every
// non-sensitive token byte-for-byte. Quoted values follow conninfo
// rules: single quotes with backslash escapes. Anything malformed
// (unterminated quotes, stray characters, tokens without "=") fails
// closed to full redaction.
func redactKeywords(dsn string) (string, bool) {
	var out []string
	i, n := 0, len(dsn)
	for i < n {
		for i < n && isSpace(dsn[i]) {
			i++
		}
		if i >= n {
			break
		}
		tokStart := i
		for i < n && dsn[i] != '=' && !isSpace(dsn[i]) {
			i++
		}
		if i >= n || dsn[i] != '=' {
			return "", false
		}
		key := dsn[tokStart:i]
		if key == "" {
			return "", false
		}
		i++ // consume '='
		for i < n && isSpace(dsn[i]) {
			i++
		}
		if i < n && dsn[i] == '\'' {
			i++
			closed := false
			for i < n {
				c := dsn[i]
				if c == '\\' {
					i += 2
					if i > n {
						return "", false
					}
					continue
				}
				if c == '\'' {
					closed = true
					i++
					break
				}
				i++
			}
			if !closed {
				return "", false
			}
			if i < n && !isSpace(dsn[i]) {
				return "", false
			}
		} else {
			for i < n && !isSpace(dsn[i]) {
				if dsn[i] == '\'' {
					return "", false
				}
				i++
			}
		}
		switch strings.ToLower(key) {
		case "password", "passfile":
			out = append(out, key+"=***")
		default:
			out = append(out, dsn[tokStart:i])
		}
	}
	return strings.Join(out, " "), true
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
