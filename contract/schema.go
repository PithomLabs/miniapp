package contract

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
)

// SchemaTargets maps committed schema file names to the struct that
// generates them. Every frozen contract shape with a JSON surface
// appears here exactly once.
func SchemaTargets() map[string]any {
	return map[string]any{
		"miniapp.schema.json":    &MiniappManifest{},
		"capability.schema.json": &CapabilitySpec{},
		"profile.schema.json":    &DeployProfile{},
		"caller.schema.json":     &Caller{},
		"run.schema.json":        &RunHandle{},
		"effect.schema.json":     &EffectRecord{},
		"approval.schema.json":   &ApprovalRecord{},
		"task.schema.json":       &TaskRecord{},
		"promotion.schema.json":  &PromotionRecord{},
		"release.schema.json":    &ReleaseEnvelope{},
		"untrusted.schema.json":  &UntrustedText{},
	}
}

// GenerateSchemas writes one JSON Schema per target into dir, using
// stable indentation so output is byte-deterministic for drift tests.
func GenerateSchemas(dir string) error {
	r := new(jsonschema.Reflector)
	for name, target := range SchemaTargets() {
		schema := r.Reflect(target)
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return Terminal("schemas", "cannot marshal %s: %v", name, err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return Terminal("schemas", "cannot write %s: %v", name, err)
		}
	}
	return nil
}
