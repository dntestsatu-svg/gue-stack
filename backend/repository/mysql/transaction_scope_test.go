package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionVisibilityCTE_RoleSpecificScope(t *testing.T) {
	query := transactionVisibilityCTE()

	require.True(t, strings.Contains(query, "au.role IN ('dev', 'superadmin')"))
	require.True(t, strings.Contains(query, "(au.role = 'admin' AND tu.user_id = au.id)"))
	require.True(t, strings.Contains(query, "(au.role = 'user' AND au.created_by IS NOT NULL AND tu.user_id = au.created_by)"))
}
