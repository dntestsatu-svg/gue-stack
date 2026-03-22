package mysql

import "github.com/example/gue/backend/model"

func canViewAllTokos(role model.UserRole) bool {
	return role == model.UserRoleDev || role == model.UserRoleSuperAdmin
}

func tokoVisibilityCTE() string {
	return `
WITH actor_scope AS (
  SELECT id
  FROM users
  WHERE id = ?
  UNION
  SELECT created_by
  FROM users
  WHERE id = ? AND created_by IS NOT NULL
),
accessible_tokos AS (
  SELECT DISTINCT tu.toko_id
  FROM toko_users tu
  INNER JOIN actor_scope scope ON scope.id = tu.user_id
)
`
}
