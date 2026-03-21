package service

import (
	"context"
	"testing"
	"time"

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

func (f *fakeTokoDomainRepo) ListByUser(_ context.Context, _ uint64) ([]model.Toko, error) {
	return nil, nil
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

func (f *fakeTokoDomainRepo) GetByToken(_ context.Context, _ string) (*model.Toko, error) {
	return nil, repository.ErrNotFound
}

type fakeBalanceRepo struct {
	byTokoID map[uint64]repository.TokoBalanceRecord
}

func (f *fakeBalanceRepo) ListByUser(_ context.Context, _ uint64) ([]repository.TokoBalanceRecord, error) {
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

func TestTokoServiceCreateForUserQuotaLimit(t *testing.T) {
	repo := &fakeTokoDomainRepo{countByUser: 3}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, 3, 3)

	_, err := svc.CreateForUser(context.Background(), 10, CreateTokoInput{
		Name: "Toko A",
	})

	require.Error(t, err)
	require.Len(t, repo.created, 0)
}

func TestTokoServiceCreateForUserSuccess(t *testing.T) {
	repo := &fakeTokoDomainRepo{countByUser: 1}
	svc := NewTokoService(repo, &fakeBalanceRepo{}, 3, 3)

	result, err := svc.CreateForUser(context.Background(), 10, CreateTokoInput{
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
				AvailableBalance:   0,
				LastSettlementTime: time.Now().UTC(),
			},
		},
	}
	svc := NewTokoService(tokoRepo, balanceRepo, 3, 3)

	result, err := svc.ManualSettlement(context.Background(), model.UserRoleDev, 12, ManualSettlementInput{
		SettlementBalance: 250000,
		AvailableBalance:  100000,
	})

	require.NoError(t, err)
	require.Equal(t, uint64(12), result.TokoID)
	require.Equal(t, 250000.0, result.SettlementBalance)
	require.Equal(t, 100000.0, result.AvailableBalance)
}
