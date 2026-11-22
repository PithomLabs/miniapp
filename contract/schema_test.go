package contract

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestSchemasCurrent regenerates every JSON Schema and requires the
// committed copies under schemas/ to match byte for byte. A mismatch
// means the structs changed without regenerating (or vice versa).
func TestSchemasCurrent(t *testing.T) {
	tmp := t.TempDir()
	if err := GenerateSchemas(tmp); err != nil {
		t.Fatal(err)
	}
	for name := range SchemaTargets() {
		want, err := os.ReadFile(filepath.Join("schemas", name))
		if err != nil {
			t.Fatalf("committed schema %s missing: %v (run GenerateSchemas)", name, err)
		}
		got, err := os.ReadFile(filepath.Join(tmp, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("schema drift in %s: regenerate schemas/", name)
		}
	}
}
