package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionVisibilityCTE_RoleSpecificScope(t *testing.T) {
	query := transactionVisibilityCTE()

	require.True(t, strings.Contains(query, "WITH RECURSIVE actor_user AS"))
	require.True(t, strings.Contains(query, "INNER JOIN hierarchy h ON u.created_by = h.id"))
	require.True(t, strings.Contains(query, "WHERE role = 'user' AND created_by IS NOT NULL"))
	require.True(t, strings.Contains(query, "WHERE au.role = 'dev' OR su.id IS NOT NULL"))
}
