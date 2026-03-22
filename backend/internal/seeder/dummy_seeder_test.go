package seeder

import (
	"testing"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/stretchr/testify/require"
)

func TestBuildDummySeedPlanHonorsHierarchyAndQuota(t *testing.T) {
	t.Parallel()

	opts := DummySeedOptions{
		Password:               "SeedPassword123!",
		AdminCount:             50,
		MaxEmployeesPerAdmin:   4,
		MaxTokosPerAdmin:       3,
		TransactionsPerTokoMin: 5,
		TransactionsPerTokoMax: 5,
		RandomSeed:             42,
		BaseTime:               time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
		Domain:                 "seed.gue.local",
	}

	plan := buildDummySeedPlan(opts)

	require.Equal(t, model.UserRoleSuperAdmin, plan.SuperAdmin.Role)
	require.Equal(t, "superadmin@seed.gue.local", plan.SuperAdmin.Email)
	require.Len(t, plan.Admins, 50)

	seenEmails := map[string]struct{}{
		plan.SuperAdmin.Email: {},
	}
	seenTokens := make(map[string]struct{})
	totalEmployees := 0
	totalTokos := 0
	totalTransactions := 0

	for _, admin := range plan.Admins {
		require.Equal(t, model.UserRoleAdmin, admin.User.Role)
		require.NotEmpty(t, admin.User.Email)
		requireEmailUnique(t, seenEmails, admin.User.Email)

		require.GreaterOrEqual(t, len(admin.Tokos), 1)
		require.LessOrEqual(t, len(admin.Tokos), 3)

		for _, employee := range admin.Employees {
			require.Equal(t, model.UserRoleUser, employee.Role)
			requireEmailUnique(t, seenEmails, employee.Email)
			totalEmployees++
		}

		for _, toko := range admin.Tokos {
			require.NotEmpty(t, toko.Token)
			if _, exists := seenTokens[toko.Token]; exists {
				t.Fatalf("duplicate toko token generated: %s", toko.Token)
			}
			seenTokens[toko.Token] = struct{}{}

			require.NotNil(t, toko.CallbackURL)
			require.GreaterOrEqual(t, toko.AvailableBalance.IntPart(), int64(0))
			require.GreaterOrEqual(t, toko.SettlementBalance.IntPart(), int64(0))
			require.Len(t, toko.Transactions, 5)

			for _, trx := range toko.Transactions {
				require.NotNil(t, trx.Reference)
				require.NotNil(t, trx.Player)
				require.NotNil(t, trx.Code)
				require.False(t, trx.CreatedAt.IsZero())
				require.False(t, trx.UpdatedAt.IsZero())
			}

			totalTokos++
			totalTransactions += len(toko.Transactions)
		}
	}

	require.GreaterOrEqual(t, totalEmployees, 0)
	require.Greater(t, totalTokos, 0)
	require.Equal(t, totalTokos*5, totalTransactions)
}

func TestValidateDummySeedOptionsRejectsInvalidRange(t *testing.T) {
	t.Parallel()

	err := validateDummySeedOptions(DummySeedOptions{
		Password:               "x",
		AdminCount:             1,
		MaxEmployeesPerAdmin:   1,
		MaxTokosPerAdmin:       1,
		TransactionsPerTokoMin: 5,
		TransactionsPerTokoMax: 1,
		RandomSeed:             1,
		BaseTime:               time.Now().UTC(),
		Domain:                 "seed.gue.local",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "maximum transactions per toko")
}

func requireEmailUnique(t *testing.T, seen map[string]struct{}, email string) {
	t.Helper()

	if _, exists := seen[email]; exists {
		t.Fatalf("duplicate email generated: %s", email)
	}
	seen[email] = struct{}{}
}
