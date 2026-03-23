package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
)

type fakeTokoDomainRepo struct {
	countByUser int
	nextID      uint64
	created     []*model.Toko
	attached    []struct {
		userID uint64
		tokoID uint64
	}
	byID map[uint64]*model.Toko

	workspaceItems        []repository.TokoWorkspaceRecord
	workspaceSummary      repository.TokoWorkspaceSummary
	workspaceListCalls    int
	workspaceSummaryCalls int
}

func (f *fakeTokoDomainRepo) Create(_ context.Context, toko *model.Toko) error {
	f.nextID++
	toko.ID = f.nextID
	f.created = append(f.created, toko)
	if f.byID == nil {
		f.byID = map[uint64]*model.Toko{}
	}
	f.byID[toko.ID] = toko
	return nil
}

func (f *fakeTokoDomainRepo) CreateForUserWithQuota(ctx context.Context, userID uint64, toko *model.Toko, maxTokos int) error {
	if maxTokos <= 0 {
		maxTokos = 3
	}
	if f.countByUser >= maxTokos {
		return repository.ErrQuotaExceeded
	}
	if err := f.Create(ctx, toko); err != nil {
		return err
	}
	if err := f.AttachUser(ctx, userID, toko.ID); err != nil {
		return err
	}
	f.countByUser++
	return nil
}

func (f *fakeTokoDomainRepo) AttachUser(_ context.Context, userID, tokoID uint64) error {
	f.attached = append(f.attached, struct {
		userID uint64
		tokoID uint64
	}{userID: userID, tokoID: tokoID})
	return nil
}

func (f *fakeTokoDomainRepo) CountByUser(_ context.Context, _ uint64) (int, error) {
	return f.countByUser, nil
}

func (f *fakeTokoDomainRepo) ListByUser(_ context.Context, _ uint64, _ model.UserRole) ([]model.Toko, error) {
	return nil, nil
}

func (f *fakeTokoDomainRepo) ListWorkspaceByUser(_ context.Context, _ uint64, _ model.UserRole, filter repository.TokoWorkspaceFilter) ([]repository.TokoWorkspaceRecord, error) {
	f.workspaceListCalls++
	if filter.Offset >= len(f.workspaceItems) {
		return []repository.TokoWorkspaceRecord{}, nil
	}

	end := filter.Offset + filter.Limit
	if end > len(f.workspaceItems) {
		end = len(f.workspaceItems)
	}
	return append([]repository.TokoWorkspaceRecord(nil), f.workspaceItems[filter.Offset:end]...), nil
}

func (f *fakeTokoDomainRepo) SummarizeWorkspaceByUser(_ context.Context, _ uint64, _ model.UserRole, _ repository.TokoWorkspaceFilter) (*repository.TokoWorkspaceSummary, error) {
	f.workspaceSummaryCalls++
	summary := f.workspaceSummary
	return &summary, nil
}

func (f *fakeTokoDomainRepo) GetByID(_ context.Context, id uint64) (*model.Toko, error) {
	if f.byID == nil {
		return nil, repository.ErrNotFound
	}
	item, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return item, nil
}

func (f *fakeTokoDomainRepo) GetAccessibleByID(ctx context.Context, _ uint64, _ model.UserRole, tokoID uint64) (*model.Toko, error) {
	return f.GetByID(ctx, tokoID)
}

func (f *fakeTokoDomainRepo) GetByToken(_ context.Context, _ string) (*model.Toko, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeTokoDomainRepo) UpdateProfile(_ context.Context, tokoID uint64, name string, callbackURL *string) error {
	if f.byID == nil {
		return repository.ErrNotFound
	}
	item, ok := f.byID[tokoID]
	if !ok {
		return repository.ErrNotFound
	}
	item.Name = name
	item.CallbackURL = callbackURL
	return nil
}

func (f *fakeTokoDomainRepo) UpdateToken(_ context.Context, tokoID uint64, token string) error {
	if f.byID == nil {
		return repository.ErrNotFound
	}
	item, ok := f.byID[tokoID]
	if !ok {
		return repository.ErrNotFound
	}
	item.Token = token
	return nil
}

type fakeBalanceRepo struct {
	byTokoID map[uint64]repository.TokoBalanceRecord
}

func (f *fakeBalanceRepo) ListByUser(_ context.Context, _ uint64, _ model.UserRole) ([]repository.TokoBalanceRecord, error) {
	result := make([]repository.TokoBalanceRecord, 0, len(f.byTokoID))
	for _, item := range f.byTokoID {
		result = append(result, item)
	}
	return result, nil
}

func (f *fakeBalanceRepo) GetByTokoID(_ context.Context, tokoID uint64) (*repository.TokoBalanceRecord, error) {
	item, ok := f.byTokoID[tokoID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copy := item
	return &copy, nil
}

func (f *fakeBalanceRepo) UpsertByTokoID(_ context.Context, tokoID uint64, settlementBalance float64, availableBalance float64) error {
	if f.byTokoID == nil {
		f.byTokoID = map[uint64]repository.TokoBalanceRecord{}
	}
	item := f.byTokoID[tokoID]
	item.TokoID = tokoID
	item.SettlementBalance = settlementBalance
	item.AvailableBalance = availableBalance
	item.LastSettlementTime = time.Now().UTC()
	f.byTokoID[tokoID] = item
	return nil
}

func (f *fakeBalanceRepo) DecreaseSettlementByTokoID(_ context.Context, tokoID uint64, amount float64) error {
	item, ok := f.byTokoID[tokoID]
	if !ok {
		return repository.ErrNotFound
	}
	if item.SettlementBalance < amount {
		return repository.ErrInsufficientBalance
	}
	item.SettlementBalance -= amount
	item.LastSettlementTime = time.Now().UTC()
	f.byTokoID[tokoID] = item
	return nil
}

func (f *fakeBalanceRepo) IncreaseSettlementByTokoID(_ context.Context, tokoID uint64, amount float64) error {
	item, ok := f.byTokoID[tokoID]
	if !ok {
		return repository.ErrNotFound
	}
	item.SettlementBalance += amount
	item.LastSettlementTime = time.Now().UTC()
	f.byTokoID[tokoID] = item
	return nil
}

func TestTokoServiceCreateForUserQuotaLimit(t *testing.T) {
	repo := &fakeTokoDomainRepo{countByUser: 3}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	_, err := svc.CreateForUser(context.Background(), 10, model.UserRoleAdmin, CreateTokoInput{
		Name: "Toko A",
	})

	require.Error(t, err)
	require.Len(t, repo.created, 0)
}

func TestTokoServiceCreateForUserSuccess(t *testing.T) {
	repo := &fakeTokoDomainRepo{countByUser: 1}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	result, err := svc.CreateForUser(context.Background(), 10, model.UserRoleAdmin, CreateTokoInput{
		Name: "Toko Alpha",
	})

	require.NoError(t, err)
	require.NotZero(t, result.ID)
	require.NotEmpty(t, result.Token)
	require.Equal(t, 3, result.Charge)
	require.Len(t, repo.created, 1)
	require.Len(t, repo.attached, 1)
	require.Equal(t, uint64(10), repo.attached[0].userID)
	require.Equal(t, result.ID, repo.attached[0].tokoID)
}

func TestTokoServiceCreateForUserForbiddenForUserRole(t *testing.T) {
	repo := &fakeTokoDomainRepo{countByUser: 0}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	_, err := svc.CreateForUser(context.Background(), 10, model.UserRoleUser, CreateTokoInput{
		Name: "Toko Forbidden",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestTokoServiceManualSettlementSuccess(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			12: {ID: 12, Name: "Toko Delta"},
		},
	}
	balanceRepo := &fakeBalanceRepo{
		byTokoID: map[uint64]repository.TokoBalanceRecord{
			12: {
				TokoID:             12,
				TokoName:           "Toko Delta",
				SettlementBalance:  0,
				AvailableBalance:   500000,
				LastSettlementTime: time.Now().UTC(),
			},
		},
	}
	svc := NewTokoService(tokoRepo, balanceRepo, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	result, err := svc.ManualSettlement(context.Background(), model.UserRoleDev, 12, ManualSettlementInput{
		SettlementBalance: 250000,
	})

	require.NoError(t, err)
	require.Equal(t, uint64(12), result.TokoID)
	require.Equal(t, 250000.0, result.SettlementBalance)
	require.Equal(t, 247000.0, result.AvailableBalance)
}

func TestTokoServiceManualSettlementAccumulatesSettlementAndDeductsFee(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			99: {ID: 99, Name: "Toko Omega"},
		},
	}
	balanceRepo := &fakeBalanceRepo{
		byTokoID: map[uint64]repository.TokoBalanceRecord{
			99: {
				TokoID:             99,
				TokoName:           "Toko Omega",
				SettlementBalance:  10000,
				AvailableBalance:   50000,
				LastSettlementTime: time.Now().UTC(),
			},
		},
	}
	svc := NewTokoService(tokoRepo, balanceRepo, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	result, err := svc.ManualSettlement(context.Background(), model.UserRoleDev, 99, ManualSettlementInput{
		SettlementBalance: 12000,
	})

	require.NoError(t, err)
	require.Equal(t, 22000.0, result.SettlementBalance)
	require.Equal(t, 35000.0, result.AvailableBalance)
}

func TestTokoServiceManualSettlementForbiddenForSuperAdmin(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			12: {ID: 12, Name: "Toko Delta"},
		},
	}
	svc := NewTokoService(tokoRepo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	_, err := svc.ManualSettlement(context.Background(), model.UserRoleSuperAdmin, 12, ManualSettlementInput{
		SettlementBalance: 1000,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "manual settlement only allowed for dev")
}

func TestTokoServiceManualSettlementRejectsNegativeAvailable(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			50: {ID: 50, Name: "Toko Sigma"},
		},
	}
	balanceRepo := &fakeBalanceRepo{
		byTokoID: map[uint64]repository.TokoBalanceRecord{
			50: {
				TokoID:             50,
				TokoName:           "Toko Sigma",
				SettlementBalance:  0,
				AvailableBalance:   5000,
				LastSettlementTime: time.Now().UTC(),
			},
		},
	}
	svc := NewTokoService(tokoRepo, balanceRepo, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	_, err := svc.ManualSettlement(context.Background(), model.UserRoleDev, 50, ManualSettlementInput{
		SettlementBalance: 4000,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient available balance")
}

func TestTokoServiceWorkspaceReturnsPaginatedRows(t *testing.T) {
	repo := &fakeTokoDomainRepo{
		workspaceItems: []repository.TokoWorkspaceRecord{
			{
				ID:                 1,
				Name:               "Toko Alpha",
				Token:              "tok_alpha",
				Charge:             3,
				SettlementBalance:  120000,
				AvailableBalance:   450000,
				LastSettlementTime: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:                 2,
				Name:               "Toko Beta",
				Token:              "tok_beta",
				Charge:             3,
				SettlementBalance:  80000,
				AvailableBalance:   250000,
				LastSettlementTime: time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC),
			},
		},
		workspaceSummary: repository.TokoWorkspaceSummary{
			TotalTokos:            2,
			TotalSettlementAmount: 200000,
			TotalAvailableAmount:  700000,
		},
	}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	page, err := svc.Workspace(context.Background(), 10, model.UserRoleAdmin, TokoWorkspaceQuery{
		Limit:  1,
		Offset: 0,
	})

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, uint64(2), page.Total)
	require.Equal(t, uint64(2), page.Summary.TotalTokos)
	require.Equal(t, 200000.0, page.Summary.TotalSettlementAmount)
	require.Equal(t, 700000.0, page.Summary.TotalAvailableAmount)
	require.True(t, page.HasMore)
}

func TestTokoServiceWorkspaceUsesCacheForPaginatedResult(t *testing.T) {
	repo := &fakeTokoDomainRepo{
		workspaceItems: []repository.TokoWorkspaceRecord{
			{
				ID:                 1,
				Name:               "Toko Alpha",
				Token:              "tok_alpha",
				Charge:             3,
				SettlementBalance:  120000,
				AvailableBalance:   450000,
				LastSettlementTime: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			},
		},
		workspaceSummary: repository.TokoWorkspaceSummary{
			TotalTokos:            1,
			TotalSettlementAmount: 120000,
			TotalAvailableAmount:  450000,
		},
	}
	cacheStore := newFakeCacheStore()
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cacheStore, true, time.Minute, 3, 3, slog.Default())
	query := TokoWorkspaceQuery{Limit: 10, Offset: 0}

	firstPage, err := svc.Workspace(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 1)
	require.Equal(t, 1, repo.workspaceSummaryCalls)
	require.Equal(t, 1, repo.workspaceListCalls)

	secondPage, err := svc.Workspace(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 1)
	require.Equal(t, 1, repo.workspaceSummaryCalls)
	require.Equal(t, 1, repo.workspaceListCalls)
}

func TestTokoServiceCreateInvalidatesWorkspaceNamespace(t *testing.T) {
	repo := &fakeTokoDomainRepo{
		countByUser: 0,
		workspaceItems: []repository.TokoWorkspaceRecord{
			{
				ID:                 1,
				Name:               "Toko Alpha",
				Token:              "tok_alpha",
				Charge:             3,
				SettlementBalance:  120000,
				AvailableBalance:   450000,
				LastSettlementTime: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			},
		},
		workspaceSummary: repository.TokoWorkspaceSummary{
			TotalTokos:            1,
			TotalSettlementAmount: 120000,
			TotalAvailableAmount:  450000,
		},
	}
	cacheStore := newFakeCacheStore()
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cacheStore, true, time.Minute, 3, 3, slog.Default())
	query := TokoWorkspaceQuery{Limit: 10, Offset: 0}

	_, err := svc.Workspace(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Equal(t, 1, repo.workspaceSummaryCalls)
	require.Equal(t, 1, repo.workspaceListCalls)

	_, err = svc.CreateForUser(context.Background(), 10, model.UserRoleAdmin, CreateTokoInput{Name: "Toko Baru"})
	require.NoError(t, err)

	repo.workspaceItems = append(repo.workspaceItems, repository.TokoWorkspaceRecord{
		ID:                 2,
		Name:               "Toko Baru",
		Token:              "tok_baru",
		Charge:             3,
		SettlementBalance:  0,
		AvailableBalance:   0,
		LastSettlementTime: time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
	})
	repo.workspaceSummary = repository.TokoWorkspaceSummary{
		TotalTokos:            2,
		TotalSettlementAmount: 120000,
		TotalAvailableAmount:  450000,
	}

	page, err := svc.Workspace(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, 2, repo.workspaceSummaryCalls)
	require.Equal(t, 2, repo.workspaceListCalls)
}

func TestTokoServiceUpdateSuccess(t *testing.T) {
	callback := "https://example.com/callback"
	repo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			7: {ID: 7, Name: "Toko Lama", Token: "token-lama", Charge: 3, CallbackURL: &callback},
		},
	}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	newCallback := "https://example.com/baru"
	updated, err := svc.Update(context.Background(), 10, model.UserRoleAdmin, 7, UpdateTokoInput{
		Name:        "Toko Baru",
		CallbackURL: &newCallback,
	})

	require.NoError(t, err)
	require.Equal(t, "Toko Baru", updated.Name)
	require.NotNil(t, updated.CallbackURL)
	require.Equal(t, newCallback, *updated.CallbackURL)
}

func TestTokoServiceUpdateForbiddenForUserRole(t *testing.T) {
	repo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			7: {ID: 7, Name: "Toko Lama", Token: "token-lama", Charge: 3},
		},
	}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	_, err := svc.Update(context.Background(), 10, model.UserRoleUser, 7, UpdateTokoInput{
		Name: "Toko Baru",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestTokoServiceRegenerateTokenSuccess(t *testing.T) {
	repo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			7: {ID: 7, Name: "Toko Alpha", Token: "token-lama", Charge: 3},
		},
	}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, cache.NewNoopCache(), false, time.Minute, 3, 3, slog.Default())

	updated, err := svc.RegenerateToken(context.Background(), 10, model.UserRoleAdmin, 7)

	require.NoError(t, err)
	require.NotEmpty(t, updated.Token)
	require.NotEqual(t, "token-lama", updated.Token)
	require.Equal(t, updated.Token, repo.byID[7].Token)
}
