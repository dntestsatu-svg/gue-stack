package mysql

import (
	"strings"
	"testing"

	"github.com/example/gue/backend/model"
	"github.com/stretchr/testify/require"
)

func TestCanViewAllTokos_OnlyDevHasGlobalVisibility(t *testing.T) {
	require.True(t, canViewAllTokos(model.UserRoleDev))
	require.False(t, canViewAllTokos(model.UserRoleSuperAdmin))
	require.False(t, canViewAllTokos(model.UserRoleAdmin))
	require.False(t, canViewAllTokos(model.UserRoleUser))
}

func TestTokoVisibilityCTE_HierarchyAwareScope(t *testing.T) {
	query := tokoVisibilityCTE()

	require.True(t, strings.Contains(query, "WITH RECURSIVE actor_user AS"))
	require.True(t, strings.Contains(query, "INNER JOIN hierarchy h ON u.created_by = h.id"))
	require.True(t, strings.Contains(query, "WHERE au.role = 'user' AND au.created_by IS NOT NULL"))
	require.True(t, strings.Contains(query, "WHERE au.role = 'dev' OR su.id IS NOT NULL"))
}
