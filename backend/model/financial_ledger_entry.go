package model

import "time"

type FinancialLedgerEntryType string

const (
	FinancialLedgerEntryDepositPendingCredit       FinancialLedgerEntryType = "deposit_pending_credit"
	FinancialLedgerEntryProjectPlatformFeeCredit   FinancialLedgerEntryType = "project_platform_fee_credit"
	FinancialLedgerEntryManualSettlementPendingDeb FinancialLedgerEntryType = "manual_settlement_pending_debit"
	FinancialLedgerEntryManualSettlementSettleCred FinancialLedgerEntryType = "manual_settlement_settle_credit"
	FinancialLedgerEntryWithdrawSettleDebit        FinancialLedgerEntryType = "withdraw_settle_debit"
	FinancialLedgerEntryWithdrawFeeDebit           FinancialLedgerEntryType = "withdraw_fee_debit"
	FinancialLedgerEntryWithdrawSettleRefund       FinancialLedgerEntryType = "withdraw_settle_refund"
	FinancialLedgerEntryWithdrawFeeRefund          FinancialLedgerEntryType = "withdraw_fee_refund"
)

type FinancialLedgerEntry struct {
	ID            uint64                   `json:"id" gorm:"column:id;primaryKey;autoIncrement;type:bigint unsigned"`
	TokoID        *uint64                  `json:"toko_id,omitempty" gorm:"column:toko_id;type:bigint unsigned;index:idx_financial_ledger_toko_id"`
	TransactionID *uint64                  `json:"transaction_id,omitempty" gorm:"column:transaction_id;type:bigint unsigned;index:idx_financial_ledger_transaction_id"`
	EntryType     FinancialLedgerEntryType `json:"entry_type" gorm:"column:entry_type;type:varchar(64);not null;index:idx_financial_ledger_entry_type"`
	Amount        uint64                   `json:"amount" gorm:"column:amount;type:bigint unsigned;not null"`
	Reference     *string                  `json:"reference,omitempty" gorm:"column:reference;type:varchar(255);index:idx_financial_ledger_reference"`
	CreatedAt     time.Time                `json:"created_at"`
}

func (FinancialLedgerEntry) TableName() string {
	return "financial_ledger_entries"
}
