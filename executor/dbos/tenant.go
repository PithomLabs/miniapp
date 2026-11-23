package dbosexec

import (
	"context"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
)

// ArmTenant sets the RLS tenant context inside a workflow
// transaction step. Workflow steps call it first, with the tenant
// carried in the translated workflow input, so every durable write
// lands under the invoking tenant even across crash recovery.
func ArmTenant(ctx context.Context, tx dbos.Tx, tenantID string) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantID)
	return err
}
