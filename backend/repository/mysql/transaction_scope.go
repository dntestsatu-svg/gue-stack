package mysql

func transactionVisibilityCTE() string {
	return `
WITH actor_user AS (
  SELECT id, role, created_by
  FROM users
  WHERE id = ?
),
accessible_tokos AS (
  SELECT DISTINCT tu.toko_id
  FROM toko_users tu
  CROSS JOIN actor_user au
  WHERE au.role IN ('dev', 'superadmin')
     OR (au.role = 'admin' AND tu.user_id = au.id)
     OR (au.role = 'user' AND au.created_by IS NOT NULL AND tu.user_id = au.created_by)
)
`
}
