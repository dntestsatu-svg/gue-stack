package mysql

func transactionVisibilityCTE() string {
	return `
WITH RECURSIVE actor_user AS (
  SELECT id, role, created_by
  FROM users
  WHERE id = ?
),
hierarchy AS (
  SELECT id
  FROM actor_user
  UNION ALL
  SELECT u.id
  FROM users u
  INNER JOIN hierarchy h ON u.created_by = h.id
),
scoped_users AS (
  SELECT id
  FROM hierarchy
  UNION
  SELECT created_by
  FROM actor_user
  WHERE role = 'user' AND created_by IS NOT NULL
),
accessible_tokos AS (
  SELECT DISTINCT tu.toko_id
  FROM toko_users tu
  CROSS JOIN actor_user au
  LEFT JOIN scoped_users su ON su.id = tu.user_id
  WHERE au.role = 'dev' OR su.id IS NOT NULL
)
`
}
