package repository

import (
	"context"
	"time"

	"github.com/example/gue/backend/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uint64) (*model.User, error)
	ListByScope(ctx context.Context, actorUserID uint64, limit int) ([]model.User, error)
	ListPageByScope(ctx context.Context, actorUserID uint64, filter UserListFilter) ([]model.User, error)
	CountByScope(ctx context.Context, actorUserID uint64, filter UserListFilter) (uint64, error)
	IsInScope(ctx context.Context, actorUserID uint64, targetUserID uint64) (bool, error)
	UpdateRole(ctx context.Context, id uint64, role model.UserRole) error
	UpdateActive(ctx context.Context, id uint64, isActive bool) error
	UpdatePassword(ctx context.Context, id uint64, passwordHash string) error
	Delete(ctx context.Context, id uint64) error
}

type TokoRepository interface {
	Create(ctx context.Context, toko *model.Toko) error
	CreateForUserWithQuota(ctx context.Context, userID uint64, toko *model.Toko, maxTokos int) error
	AttachUser(ctx context.Context, userID, tokoID uint64) error
	CountByUser(ctx context.Context, userID uint64) (int, error)
	ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]model.Toko, error)
	ListWorkspaceByUser(ctx context.Context, userID uint64, actorRole model.UserRole, filter TokoWorkspaceFilter) ([]TokoWorkspaceRecord, error)
	SummarizeWorkspaceByUser(ctx context.Context, userID uint64, actorRole model.UserRole, filter TokoWorkspaceFilter) (*TokoWorkspaceSummary, error)
	GetByID(ctx context.Context, id uint64) (*model.Toko, error)
	GetAccessibleByID(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64) (*model.Toko, error)
	GetByToken(ctx context.Context, token string) (*model.Toko, error)
	UpdateProfile(ctx context.Context, tokoID uint64, name string, callbackURL *string) error
	UpdateToken(ctx context.Context, tokoID uint64, token string) error
}

type BalanceRepository interface {
	ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]TokoBalanceRecord, error)
	GetByTokoID(ctx context.Context, tokoID uint64) (*TokoBalanceRecord, error)
	UpsertByTokoID(ctx context.Context, tokoID uint64, settlementBalance float64, availableBalance float64) error
	DecreaseSettlementByTokoID(ctx context.Context, tokoID uint64, amount float64) error
	IncreaseSettlementByTokoID(ctx context.Context, tokoID uint64, amount float64) error
}

type BankRepository interface {
	ListByUser(ctx context.Context, userID uint64, filter BankListFilter) ([]model.Bank, error)
	CountByUser(ctx context.Context, userID uint64, filter BankListFilter) (uint64, error)
	GetByUser(ctx context.Context, userID uint64, bankID uint64) (*model.Bank, error)
	Create(ctx context.Context, bank *model.Bank) error
	DeleteByUser(ctx context.Context, userID uint64, bankID uint64) error
}

type PaymentRepository interface {
	GetByID(ctx context.Context, id uint64) (*model.Payment, error)
	SearchOptions(ctx context.Context, filter PaymentOptionFilter) ([]model.Payment, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, trx *model.Transaction) error
	GetByReference(ctx context.Context, reference string) (*model.Transaction, error)
	GetByReferenceAndToko(ctx context.Context, reference string, tokoID uint64) (*model.Transaction, error)
	UpdateStatusByReference(ctx context.Context, reference string, status model.TransactionStatus) error
	UpdateStatusByReferenceAndToko(ctx context.Context, reference string, tokoID uint64, status model.TransactionStatus) error
	UpdateSettlementByID(ctx context.Context, id uint64, status model.TransactionStatus, platformFee uint64, netto uint64) error
	GetDashboardMetricsByUser(ctx context.Context, userID uint64, from time.Time) (*DashboardMetrics, error)
	GetHourlyStatusCountsByUser(ctx context.Context, userID uint64, from time.Time) ([]DashboardStatusSeriesPoint, error)
	ListRecentByUser(ctx context.Context, userID uint64, filter TransactionHistoryFilter) ([]TransactionHistoryRecord, error)
	ListRecentSuccessByUser(ctx context.Context, userID uint64, limit int) ([]TransactionHistoryRecord, error)
	CountHistoryByUser(ctx context.Context, userID uint64, filter TransactionHistoryFilter) (uint64, error)
}

type DashboardMetrics struct {
	TotalCount            uint64
	SuccessCount          uint64
	PendingCount          uint64
	FailedCount           uint64
	SuccessDepositAmount  uint64
	SuccessWithdrawAmount uint64
	TotalPlatformFee      uint64
}

type UserListFilter struct {
	Limit      int
	Offset     int
	SearchTerm string
	Role       model.UserRole
}

type DashboardStatusSeriesPoint struct {
	Bucket       time.Time
	SuccessCount uint64
	FailedCount  uint64
}

type TokoBalanceRecord struct {
	TokoID             uint64
	TokoName           string
	SettlementBalance  float64
	AvailableBalance   float64
	LastSettlementTime time.Time
}

type TokoWorkspaceFilter struct {
	Limit      int
	Offset     int
	SearchTerm string
}

type TokoWorkspaceRecord struct {
	ID                 uint64
	Name               string
	Token              string
	Charge             int
	CallbackURL        *string
	SettlementBalance  float64
	AvailableBalance   float64
	LastSettlementTime time.Time
}

type TokoWorkspaceSummary struct {
	TotalTokos            uint64
	TotalSettlementAmount float64
	TotalAvailableAmount  float64
}

type TransactionHistoryRecord struct {
	ID        uint64
	TokoID    uint64
	TokoName  string
	Player    *string
	Code      *string
	Type      model.TransactionType
	Status    model.TransactionStatus
	Reference *string
	Amount    uint64
	Netto     uint64
	CreatedAt time.Time
}

type TransactionHistoryFilter struct {
	Limit      int
	Offset     int
	From       *time.Time
	To         *time.Time
	SearchTerm string
	Type       model.TransactionType
}

type BankListFilter struct {
	Limit      int
	Offset     int
	SearchTerm string
}

type PaymentOptionFilter struct {
	Limit      int
	SearchTerm string
}

type RefreshTokenStore interface {
	Store(ctx context.Context, tokenID string, userID uint64, ttl time.Duration) error
	GetUserID(ctx context.Context, tokenID string) (uint64, error)
	Delete(ctx context.Context, tokenID string) error
}
