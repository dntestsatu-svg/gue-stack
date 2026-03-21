package seeder

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/gue/backend/model"
	"gorm.io/gorm"
)

type paymentSeed struct {
	BankCode      string
	BankName      string
	BankSwiftCode *string
}

func SeedPayments(ctx context.Context, db *gorm.DB) error {
	seeds := parsePaymentSeedData()
	if len(seeds) == 0 {
		return fmt.Errorf("payment seed data is empty")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing []model.Payment
		if err := tx.Select("bank_code", "bank_name", "bank_swift_code").Find(&existing).Error; err != nil {
			return fmt.Errorf("load existing payments: %w", err)
		}

		existingKeys := make(map[string]struct{}, len(existing))
		for _, row := range existing {
			existingKeys[paymentSeedKey(row.BankCode, row.BankName, row.BankSwiftCode)] = struct{}{}
		}

		toInsert := make([]model.Payment, 0, len(seeds))
		for _, seed := range seeds {
			key := paymentSeedKey(seed.BankCode, seed.BankName, seed.BankSwiftCode)
			if _, ok := existingKeys[key]; ok {
				continue
			}

			toInsert = append(toInsert, model.Payment{
				BankCode:      seed.BankCode,
				BankName:      seed.BankName,
				BankSwiftCode: seed.BankSwiftCode,
			})
			existingKeys[key] = struct{}{}
		}

		if len(toInsert) == 0 {
			return nil
		}

		if err := tx.CreateInBatches(toInsert, 200).Error; err != nil {
			return fmt.Errorf("insert payments seed: %w", err)
		}

		return nil
	})
}

func paymentSeedKey(bankCode, bankName string, swift *string) string {
	swiftValue := ""
	if swift != nil {
		swiftValue = *swift
	}
	return strings.Join([]string{bankCode, bankName, swiftValue}, "|")
}
