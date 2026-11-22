package config

import (
	"fmt"
	"os"
	"strings"
)

// Secret resolves one credential from an environment variable pair:
// NAME holds the value directly, NAME_FILE names a file holding it.
// Exactly one may be set. Surrounding whitespace is trimmed (files
// conventionally end with a newline); an empty result is an error.
// Errors name variables and paths only — values never appear.
func Secret(name, fileVar string) (string, error) {
	val, valSet := os.LookupEnv(name)
	path, pathSet := os.LookupEnv(fileVar)
	if valSet && pathSet {
		return "", fmt.Errorf("configuration error: %s and %s are both set; set exactly one", name, fileVar)
	}
	if valSet {
		return nonempty(name, strings.TrimSpace(val))
	}
	if pathSet {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("configuration error: cannot read %s file %s: %v", name, path, errKind(err))
		}
		return nonempty(name, strings.TrimSpace(string(data)))
	}
	return "", fmt.Errorf("configuration error: set %s or %s", name, fileVar)
}

// DatabaseURL resolves the application database credential.
func DatabaseURL() (string, error) {
	return Secret("DATABASE_URL", "DATABASE_URL_FILE")
}

// DBOSSystemDatabaseURL resolves the DBOS system database credential.
func DBOSSystemDatabaseURL() (string, error) {
	return Secret("DBOS_SYSTEM_DATABASE_URL", "DBOS_SYSTEM_DATABASE_URL_FILE")
}

func nonempty(name, v string) (string, error) {
	if v == "" {
		return "", fmt.Errorf("configuration error: %s resolved empty", name)
	}
	return v, nil
}

// errKind classifies file errors without leaking path contents.
func errKind(err error) string {
	if os.IsNotExist(err) {
		return "no such file"
	}
	if os.IsPermission(err) {
		return "permission denied"
	}
	return "read error"
}
