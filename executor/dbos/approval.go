package dbosexec

import (
	"context"

	"github.com/dbos-inc/dbos-transact-golang/dbos"

	"github.com/PithomLabs/miniapp/contract"
)

// ReadApprovalDecision loads one approval's status and decision
// inside a workflow transaction step. Workflows parked on the
// approval topic call it after every wake (signal, timeout, or
// recovery): the approvals table is authoritative, and the signal
// that woke them is only a notification. The caller arms the tenant
// (ArmTenant) before reading.
func ReadApprovalDecision(ctx context.Context, tx dbos.Tx, tenantID, approvalID string) (status string, decision contract.ApprovalDecision, err error) {
	var decided string
	err = tx.QueryRow(ctx, `SELECT status, decision FROM app.approvals
		WHERE id = $1::uuid AND tenant_id = $2`, approvalID, tenantID).Scan(&status, &decided)
	if err != nil {
		return "", "", err
	}
	return status, contract.ApprovalDecision(decided), nil
}
