package runtime

import (
	"encoding/json"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/PithomLabs/miniapp/contract"
)

// validator compiles tool input schemas once and validates inputs
// against them. Schemas come from frozen manifests; compilation
// happens lazily and is safe for concurrent use.
type validator struct {
	mu      sync.Mutex
	schemas map[string]*jsonschema.Schema
}

func newValidator() *validator {
	return &validator{schemas: map[string]*jsonschema.Schema{}}
}

// validate checks raw input against the tool's declared input schema.
// Schema compile failures are internal errors (the manifest passed
// structural validation, so its schemas parse); input mismatches are
// terminal caller errors that never leak the credential-adjacent
// details of the failure beyond the validator message.
func (v *validator) validate(manifest *contract.MiniappManifest, tool contract.ToolManifest, raw json.RawMessage) error {
	key := string(manifest.ID) + "@" + string(manifest.Version) + "/" + string(tool.Name)
	v.mu.Lock()
	sch, ok := v.schemas[key]
	v.mu.Unlock()
	if !ok {
		var doc any
		if err := json.Unmarshal(tool.InputSchema, &doc); err != nil {
			return retryable("schema-invalid", "tool schema does not parse: %v", err)
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource("schema.json", doc); err != nil {
			return retryable("schema-invalid", "tool schema rejected: %v", err)
		}
		var err error
		sch, err = c.Compile("schema.json")
		if err != nil {
			return retryable("schema-invalid", "tool schema rejected: %v", err)
		}
		v.mu.Lock()
		v.schemas[key] = sch
		v.mu.Unlock()
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return terminal("invalid-input", "input is not valid JSON: %v", err)
	}
	if err := sch.Validate(instance); err != nil {
		return terminal("invalid-input", "input violates the tool schema: %v", err)
	}
	return nil
}
