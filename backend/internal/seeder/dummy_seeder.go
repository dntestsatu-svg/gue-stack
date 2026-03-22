package seeder

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/password"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	defaultDummySeedPassword = "SeedPassword123!"
	defaultDummySeedDomain   = "seed.gue.local"
	seedTokoTokenPrefix      = "seed_toko_"
)

type DummySeedOptions struct {
	Password               string
	AdminCount             int
	MaxEmployeesPerAdmin   int
	MaxTokosPerAdmin       int
	TransactionsPerTokoMin int
	TransactionsPerTokoMax int
	RandomSeed             int64
	BaseTime               time.Time
	Domain                 string
}

type DummySeedReport struct {
	SuperAdminEmail  string
	AdminCount       int
	EmployeeCount    int
	TokoCount        int
	TransactionCount int
}

type dummySeedPlan struct {
	SuperAdmin dummyUserSeed
	Admins     []dummyAdminSeed
}

type dummyUserSeed struct {
	Key   string
	Name  string
	Email string
	Role  model.UserRole
}

type dummyAdminSeed struct {
	User      dummyUserSeed
	Employees []dummyUserSeed
	Tokos     []dummyTokoSeed
}

type dummyTokoSeed struct {
	Name              string
	Token             string
	Charge            int
	CallbackURL       *string
	AvailableBalance  decimal.Decimal
	SettlementBalance decimal.Decimal
	Transactions      []dummyTransactionSeed
}

type dummyTransactionSeed struct {
	Player        *string
	Code          *string
	Type          model.TransactionType
	Status        model.TransactionStatus
	Barcode       *string
	Reference     *string
	Amount        uint64
	FeeWithdrawal *uint64
	PlatformFee   uint64
	Netto         uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func DefaultDummySeedOptions() DummySeedOptions {
	return DummySeedOptions{
		Password:               defaultDummySeedPassword,
		AdminCount:             50,
		MaxEmployeesPerAdmin:   5,
		MaxTokosPerAdmin:       3,
		TransactionsPerTokoMin: 12,
		TransactionsPerTokoMax: 32,
		RandomSeed:             20260321,
		BaseTime:               time.Now().UTC().Truncate(time.Minute),
		Domain:                 defaultDummySeedDomain,
	}
}

func SeedDummyData(ctx context.Context, db *gorm.DB, opts DummySeedOptions) (DummySeedReport, error) {
	if err := validateDummySeedOptions(opts); err != nil {
		return DummySeedReport{}, err
	}

	plan := buildDummySeedPlan(opts)
	passwordHash, err := password.Hash(strings.TrimSpace(opts.Password))
	if err != nil {
		return DummySeedReport{}, fmt.Errorf("hash dummy seed password: %w", err)
	}

	var report DummySeedReport
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET time_zone = '+00:00'").Error; err != nil {
			return fmt.Errorf("set dummy seed transaction timezone to UTC: %w", err)
		}

		if err := cleanupDummyNamespace(tx, opts); err != nil {
			return err
		}

		devUser, err := findSeedOwnerDev(tx)
		if err != nil {
			return err
		}

		superadmin := model.User{
			Name:         plan.SuperAdmin.Name,
			Email:        plan.SuperAdmin.Email,
			PasswordHash: passwordHash,
			Role:         plan.SuperAdmin.Role,
			IsActive:     true,
			CreatedBy:    &devUser.ID,
		}
		if err := tx.Create(&superadmin).Error; err != nil {
			return fmt.Errorf("create dummy superadmin: %w", err)
		}

		report.SuperAdminEmail = superadmin.Email

		for _, adminSeed := range plan.Admins {
			admin := model.User{
				Name:         adminSeed.User.Name,
				Email:        adminSeed.User.Email,
				PasswordHash: passwordHash,
				Role:         adminSeed.User.Role,
				IsActive:     true,
				CreatedBy:    &superadmin.ID,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return fmt.Errorf("create dummy admin %s: %w", adminSeed.User.Email, err)
			}
			report.AdminCount++

			for _, employeeSeed := range adminSeed.Employees {
				employee := model.User{
					Name:         employeeSeed.Name,
					Email:        employeeSeed.Email,
					PasswordHash: passwordHash,
					Role:         employeeSeed.Role,
					IsActive:     true,
					CreatedBy:    &admin.ID,
				}
				if err := tx.Create(&employee).Error; err != nil {
					return fmt.Errorf("create dummy employee %s: %w", employeeSeed.Email, err)
				}
				report.EmployeeCount++
			}

			for _, tokoSeed := range adminSeed.Tokos {
				toko := model.Toko{
					Name:        tokoSeed.Name,
					Token:       tokoSeed.Token,
					Charge:      tokoSeed.Charge,
					CallbackURL: tokoSeed.CallbackURL,
				}
				if err := tx.Create(&toko).Error; err != nil {
					return fmt.Errorf("create dummy toko %s: %w", tokoSeed.Token, err)
				}

				ownerLink := model.TokoUser{
					UserID: admin.ID,
					TokoID: toko.ID,
				}
				if err := tx.Create(&ownerLink).Error; err != nil {
					return fmt.Errorf("attach dummy admin %s to toko %s: %w", adminSeed.User.Email, tokoSeed.Token, err)
				}

				balance := model.Balance{
					TokoID:    toko.ID,
					Pending:   tokoSeed.SettlementBalance,
					Available: tokoSeed.AvailableBalance,
				}
				if err := tx.Create(&balance).Error; err != nil {
					return fmt.Errorf("create dummy balance for toko %s: %w", tokoSeed.Token, err)
				}
				report.TokoCount++

				if len(tokoSeed.Transactions) == 0 {
					continue
				}

				transactions := make([]model.Transaction, 0, len(tokoSeed.Transactions))
				for _, trxSeed := range tokoSeed.Transactions {
					transactions = append(transactions, model.Transaction{
						TokoID:        toko.ID,
						Player:        trxSeed.Player,
						Code:          trxSeed.Code,
						Type:          trxSeed.Type,
						Status:        trxSeed.Status,
						Barcode:       trxSeed.Barcode,
						Reference:     trxSeed.Reference,
						Amount:        trxSeed.Amount,
						FeeWithdrawal: trxSeed.FeeWithdrawal,
						PlatformFee:   trxSeed.PlatformFee,
						Netto:         trxSeed.Netto,
						CreatedAt:     trxSeed.CreatedAt,
						UpdatedAt:     trxSeed.UpdatedAt,
					})
				}
				if err := tx.CreateInBatches(transactions, 100).Error; err != nil {
					return fmt.Errorf("create dummy transactions for toko %s: %w", tokoSeed.Token, err)
				}
				report.TransactionCount += len(transactions)
			}
		}

		return nil
	})

	if err != nil {
		return DummySeedReport{}, err
	}

	return report, nil
}

func validateDummySeedOptions(opts DummySeedOptions) error {
	if strings.TrimSpace(opts.Password) == "" {
		return fmt.Errorf("dummy seed password is required")
	}
	if opts.AdminCount <= 0 {
		return fmt.Errorf("dummy seed admin count must be greater than zero")
	}
	if opts.MaxEmployeesPerAdmin < 0 {
		return fmt.Errorf("dummy seed max employees per admin cannot be negative")
	}
	if opts.MaxTokosPerAdmin <= 0 {
		return fmt.Errorf("dummy seed max tokos per admin must be greater than zero")
	}
	if opts.TransactionsPerTokoMin < 0 {
		return fmt.Errorf("dummy seed minimum transactions per toko cannot be negative")
	}
	if opts.TransactionsPerTokoMax < opts.TransactionsPerTokoMin {
		return fmt.Errorf("dummy seed maximum transactions per toko must be greater than or equal to minimum")
	}
	if opts.BaseTime.IsZero() {
		return fmt.Errorf("dummy seed base time is required")
	}
	if strings.TrimSpace(opts.Domain) == "" {
		return fmt.Errorf("dummy seed email domain is required")
	}
	return nil
}

func cleanupDummyNamespace(tx *gorm.DB, opts DummySeedOptions) error {
	emailPattern := "%@" + strings.TrimSpace(opts.Domain)
	if err := tx.Where("token LIKE ?", seedTokoTokenPrefix+"%").Delete(&model.Toko{}).Error; err != nil {
		return fmt.Errorf("delete existing dummy tokos: %w", err)
	}
	if err := tx.Where("email LIKE ?", emailPattern).Delete(&model.User{}).Error; err != nil {
		return fmt.Errorf("delete existing dummy users: %w", err)
	}
	return nil
}

func findSeedOwnerDev(tx *gorm.DB) (*model.User, error) {
	var dev model.User
	err := tx.Where("role = ?", model.UserRoleDev).Order("id ASC").First(&dev).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("bootstrap dev user not found; run initdb without --seed first")
		}
		return nil, fmt.Errorf("load bootstrap dev user: %w", err)
	}
	return &dev, nil
}

func buildDummySeedPlan(opts DummySeedOptions) dummySeedPlan {
	rng := rand.New(rand.NewSource(opts.RandomSeed))
	domain := strings.TrimSpace(opts.Domain)

	plan := dummySeedPlan{
		SuperAdmin: dummyUserSeed{
			Key:   "superadmin",
			Name:  "Seed Superadmin",
			Email: fmt.Sprintf("superadmin@%s", domain),
			Role:  model.UserRoleSuperAdmin,
		},
		Admins: make([]dummyAdminSeed, 0, opts.AdminCount),
	}

	for adminNumber := 1; adminNumber <= opts.AdminCount; adminNumber++ {
		adminKey := fmt.Sprintf("admin%03d", adminNumber)
		adminSeed := dummyAdminSeed{
			User: dummyUserSeed{
				Key:   adminKey,
				Name:  fmt.Sprintf("Admin %03d", adminNumber),
				Email: fmt.Sprintf("%s@%s", adminKey, domain),
				Role:  model.UserRoleAdmin,
			},
			Employees: make([]dummyUserSeed, 0),
			Tokos:     make([]dummyTokoSeed, 0),
		}

		employeeCount := 0
		if opts.MaxEmployeesPerAdmin > 0 {
			employeeCount = rng.Intn(opts.MaxEmployeesPerAdmin + 1)
		}
		for employeeNumber := 1; employeeNumber <= employeeCount; employeeNumber++ {
			employeeKey := fmt.Sprintf("%s-user%02d", adminKey, employeeNumber)
			adminSeed.Employees = append(adminSeed.Employees, dummyUserSeed{
				Key:   employeeKey,
				Name:  fmt.Sprintf("User %03d-%02d", adminNumber, employeeNumber),
				Email: fmt.Sprintf("%s@%s", employeeKey, domain),
				Role:  model.UserRoleUser,
			})
		}

		tokoCount := 1
		if opts.MaxTokosPerAdmin > 1 {
			tokoCount = 1 + rng.Intn(opts.MaxTokosPerAdmin)
		}
		for tokoNumber := 1; tokoNumber <= tokoCount; tokoNumber++ {
			adminSeed.Tokos = append(adminSeed.Tokos, buildDummyTokoSeed(rng, opts, adminNumber, tokoNumber, adminKey))
		}

		plan.Admins = append(plan.Admins, adminSeed)
	}

	return plan
}

func buildDummyTokoSeed(rng *rand.Rand, opts DummySeedOptions, adminNumber, tokoNumber int, adminKey string) dummyTokoSeed {
	tokoCode := fmt.Sprintf("%s_%02d", adminKey, tokoNumber)
	callbackURL := fmt.Sprintf("https://%s.%s/callback", strings.ReplaceAll(tokoCode, "_", "-"), opts.Domain)
	transactions := buildDummyTransactions(rng, opts, adminNumber, tokoNumber, tokoCode)

	var successNetTotal uint64
	var pendingNetTotal uint64
	for _, item := range transactions {
		switch item.Status {
		case model.TransactionStatusSuccess:
			successNetTotal += item.Netto
		case model.TransactionStatusPending:
			pendingNetTotal += item.Netto
		}
	}

	available := decimal.NewFromInt(int64(successNetTotal / 3))
	if successNetTotal > 0 {
		minAvailable := int64(successNetTotal / 5)
		maxAvailable := int64(successNetTotal / 2)
		if maxAvailable > minAvailable {
			available = decimal.NewFromInt(minAvailable + int64(rng.Intn(int(maxAvailable-minAvailable)+1)))
		}
	}

	settlement := decimal.NewFromInt(int64(pendingNetTotal / 2))
	if pendingNetTotal > 0 {
		maxSettlement := int64(pendingNetTotal)
		if maxSettlement > 0 {
			settlement = decimal.NewFromInt(int64(rng.Intn(int(maxSettlement) + 1)))
		}
	}

	return dummyTokoSeed{
		Name:              buildDummyTokoName(rng, adminNumber, tokoNumber),
		Token:             seedTokoTokenPrefix + tokoCode,
		Charge:            3,
		CallbackURL:       &callbackURL,
		AvailableBalance:  available,
		SettlementBalance: settlement,
		Transactions:      transactions,
	}
}

func buildDummyTransactions(rng *rand.Rand, opts DummySeedOptions, adminNumber, tokoNumber int, tokoCode string) []dummyTransactionSeed {
	transactionCount := opts.TransactionsPerTokoMin
	if opts.TransactionsPerTokoMax > opts.TransactionsPerTokoMin {
		transactionCount += rng.Intn(opts.TransactionsPerTokoMax - opts.TransactionsPerTokoMin + 1)
	}

	transactions := make([]dummyTransactionSeed, 0, transactionCount)
	for index := 1; index <= transactionCount; index++ {
		status := pickTransactionStatus(rng)
		txType := pickTransactionType(rng)

		amount := uint64(10000 + (rng.Intn(90) * 5000))
		var feeWithdrawal *uint64
		if txType == model.TransactionTypeWithdraw {
			fee := uint64(1500 + (rng.Intn(6) * 500))
			feeWithdrawal = &fee
		}

		platformFee := uint64(0)
		if status == model.TransactionStatusSuccess {
			platformFee = amount * 3 / 100
		}

		netto := amount - platformFee
		if feeWithdrawal != nil && netto > *feeWithdrawal {
			netto -= *feeWithdrawal
		}

		player := fmt.Sprintf("player-%03d-%02d-%04d", adminNumber, tokoNumber, index)
		code := fmt.Sprintf("QR-%03d%02d%04d", adminNumber, tokoNumber, index)
		reference := fmt.Sprintf("SEED-%03d-%02d-%04d", adminNumber, tokoNumber, index)
		barcode := fmt.Sprintf("https://seed-payments.local/qr/%s/%s", strings.ToLower(strings.ReplaceAll(tokoCode, "_", "-")), strings.ToLower(reference))

		createdAt := buildSeedTimestamp(rng, opts.BaseTime)
		updatedAt := createdAt.Add(time.Duration(15+rng.Intn(165)) * time.Minute)

		transactions = append(transactions, dummyTransactionSeed{
			Player:        &player,
			Code:          &code,
			Type:          txType,
			Status:        status,
			Barcode:       &barcode,
			Reference:     &reference,
			Amount:        amount,
			FeeWithdrawal: feeWithdrawal,
			PlatformFee:   platformFee,
			Netto:         netto,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}

	return transactions
}

func pickTransactionStatus(rng *rand.Rand) model.TransactionStatus {
	roll := rng.Intn(100)
	switch {
	case roll < 65:
		return model.TransactionStatusSuccess
	case roll < 85:
		return model.TransactionStatusPending
	default:
		return model.TransactionStatusFailed
	}
}

func pickTransactionType(rng *rand.Rand) model.TransactionType {
	if rng.Intn(100) < 82 {
		return model.TransactionTypeDeposit
	}
	return model.TransactionTypeWithdraw
}

func buildDummyTokoName(rng *rand.Rand, adminNumber, tokoNumber int) string {
	prefixes := []string{"Atlas", "Nusa", "Citra", "Meraki", "Aurora", "Sagara", "Lentera", "Aruna", "Pilar", "Nexa"}
	suffixes := []string{"Mart", "Hub", "Station", "Commerce", "Outlet", "Point", "Store", "Trade", "Center", "Gateway"}

	prefix := prefixes[rng.Intn(len(prefixes))]
	suffix := suffixes[rng.Intn(len(suffixes))]
	return fmt.Sprintf("%s %s %03d-%02d", prefix, suffix, adminNumber, tokoNumber)
}

func buildSeedTimestamp(rng *rand.Rand, base time.Time) time.Time {
	dayOffset := rng.Intn(60)
	hour := 18 + rng.Intn(5)
	minute := rng.Intn(60)

	anchor := base.UTC().AddDate(0, 0, -dayOffset)
	return time.Date(anchor.Year(), anchor.Month(), anchor.Day(), hour, minute, 0, 0, time.UTC)
}
