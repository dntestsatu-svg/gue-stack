package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/money"
	"github.com/example/gue/backend/repository"
	"github.com/shopspring/decimal"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, trx *model.Transaction) error {
	query := "INSERT INTO transactions (" +
		"toko_id, player, code, `type`, status, barcode, reference, amount, fee_withdrawal, platform_fee, netto" +
		") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	result, err := r.db.ExecContext(
		ctx,
		query,
		trx.TokoID,
		nullableString(trx.Player),
		nullableString(trx.Code),
		trx.Type,
		trx.Status,
		nullableString(trx.Barcode),
		nullableString(trx.Reference),
		trx.Amount,
		nullableUint64(trx.FeeWithdrawal),
		trx.PlatformFee,
		trx.Netto,
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get transaction id: %w", err)
	}
	trx.ID = uint64(id)
	return nil
}

func (r *TransactionRepository) GetByReference(ctx context.Context, reference string) (*model.Transaction, error) {
	query := "SELECT id, toko_id, player, code, `type`, status, barcode, reference, amount, fee_withdrawal, platform_fee, netto, created_at, updated_at " +
		"FROM transactions WHERE reference = ? LIMIT 1"
	return r.getOneByQuery(ctx, query, reference)
}

func (r *TransactionRepository) GetByReferenceAndToko(ctx context.Context, reference string, tokoID uint64) (*model.Transaction, error) {
	query := "SELECT id, toko_id, player, code, `type`, status, barcode, reference, amount, fee_withdrawal, platform_fee, netto, created_at, updated_at " +
		"FROM transactions WHERE reference = ? AND toko_id = ? LIMIT 1"
	return r.getOneByQuery(ctx, query, reference, tokoID)
}

func (r *TransactionRepository) getOneByQuery(ctx context.Context, query string, args ...any) (*model.Transaction, error) {
	trx := &model.Transaction{}
	var player, code, barcode, ref sql.NullString
	var fee sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&trx.ID,
		&trx.TokoID,
		&player,
		&code,
		&trx.Type,
		&trx.Status,
		&barcode,
		&ref,
		&trx.Amount,
		&fee,
		&trx.PlatformFee,
		&trx.Netto,
		&trx.CreatedAt,
		&trx.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query transaction by reference: %w", err)
	}

	if player.Valid {
		trx.Player = &player.String
	}
	if code.Valid {
		trx.Code = &code.String
	}
	if barcode.Valid {
		trx.Barcode = &barcode.String
	}
	if ref.Valid {
		trx.Reference = &ref.String
	}
	if fee.Valid {
		v := uint64(fee.Int64)
		trx.FeeWithdrawal = &v
	}

	return trx, nil
}

func (r *TransactionRepository) UpdateStatusByReference(ctx context.Context, reference string, status model.TransactionStatus) error {
	query := `UPDATE transactions SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE reference = ?`
	result, err := r.db.ExecContext(ctx, query, status, reference)
	if err != nil {
		return fmt.Errorf("update transaction status by reference: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) UpdateStatusByReferenceAndToko(ctx context.Context, reference string, tokoID uint64, status model.TransactionStatus) error {
	query := `UPDATE transactions SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE reference = ? AND toko_id = ?`
	result, err := r.db.ExecContext(ctx, query, status, reference, tokoID)
	if err != nil {
		return fmt.Errorf("update transaction status by reference and toko: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) UpdateSettlementByID(ctx context.Context, id uint64, status model.TransactionStatus, platformFee uint64, netto uint64) error {
	query := `UPDATE transactions SET status = ?, platform_fee = ?, netto = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, status, platformFee, netto, id)
	if err != nil {
		return fmt.Errorf("update transaction settlement by id: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) UpdateSettlementIfPending(ctx context.Context, id uint64, status model.TransactionStatus, platformFee uint64, netto uint64) (bool, error) {
	query := `UPDATE transactions
SET status = ?, platform_fee = ?, netto = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = ?`
	result, err := r.db.ExecContext(ctx, query, status, platformFee, netto, id, model.TransactionStatusPending)
	if err != nil {
		return false, fmt.Errorf("update transaction settlement if pending: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected rows: %w", err)
	}
	return affected > 0, nil
}

func (r *TransactionRepository) FinalizeDepositSuccessByID(ctx context.Context, id uint64, tokoID uint64, platformFee uint64, netto uint64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin finalize deposit success transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	updateQuery := `UPDATE transactions
SET status = ?, platform_fee = ?, netto = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = ? AND type = ?`
	result, err := tx.ExecContext(
		ctx,
		updateQuery,
		model.TransactionStatusSuccess,
		platformFee,
		netto,
		id,
		model.TransactionStatusPending,
		model.TransactionTypeDeposit,
	)
	if err != nil {
		return false, fmt.Errorf("update deposit transaction status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected rows for deposit finalize: %w", err)
	}
	if rowsAffected == 0 {
		return false, nil
	}

	transactionID := id
	if err := insertFinancialLedgerEntryTx(ctx, tx, &tokoID, &transactionID, model.FinancialLedgerEntryDepositPendingCredit, netto, nil); err != nil {
		return false, err
	}
	if err := insertFinancialLedgerEntryTx(ctx, tx, &tokoID, &transactionID, model.FinancialLedgerEntryProjectPlatformFeeCredit, platformFee, nil); err != nil {
		return false, err
	}

	balanceQuery := `UPDATE balances
SET pending = pending + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ?`
	nettoValue := decimal.NewFromInt(int64(netto)).StringFixed(2)
	result, err = tx.ExecContext(ctx, balanceQuery, nettoValue, tokoID)
	if err != nil {
		return false, fmt.Errorf("increase pending balance on deposit finalize: %w", err)
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected rows for balance finalize: %w", err)
	}
	if rowsAffected == 0 {
		return false, repository.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit finalize deposit success transaction: %w", err)
	}
	committed = true
	return true, nil
}

func (r *TransactionRepository) CreatePendingWithdrawAndReserveSettlement(ctx context.Context, trx *model.Transaction) error {
	if trx == nil {
		return fmt.Errorf("create pending withdraw and reserve settlement: nil transaction")
	}
	if trx.Type != model.TransactionTypeWithdraw {
		return fmt.Errorf("create pending withdraw and reserve settlement: invalid transaction type %s", trx.Type)
	}

	fee := uint64(0)
	if trx.FeeWithdrawal != nil {
		fee = *trx.FeeWithdrawal
	}
	totalDebit, err := money.AddUint64(trx.Amount, fee)
	if err != nil {
		return fmt.Errorf("calculate withdraw total debit: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create pending withdraw transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	delta := decimal.NewFromInt(int64(totalDebit)).StringFixed(2)
	reserveQuery := `UPDATE balances
SET available = available - ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ? AND available >= ?`
	result, err := tx.ExecContext(ctx, reserveQuery, delta, trx.TokoID, delta)
	if err != nil {
		return fmt.Errorf("reserve withdraw settlement balance: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for reserve withdraw settlement balance: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrInsufficientBalance
	}

	insertQuery := "INSERT INTO transactions (" +
		"toko_id, player, code, `type`, status, barcode, reference, amount, fee_withdrawal, platform_fee, netto" +
		") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	result, err = tx.ExecContext(
		ctx,
		insertQuery,
		trx.TokoID,
		nullableString(trx.Player),
		nullableString(trx.Code),
		trx.Type,
		trx.Status,
		nullableString(trx.Barcode),
		nullableString(trx.Reference),
		trx.Amount,
		nullableUint64(trx.FeeWithdrawal),
		trx.PlatformFee,
		trx.Netto,
	)
	if err != nil {
		return fmt.Errorf("create pending withdraw transaction: %w", err)
	}
	insertedID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read pending withdraw transaction id: %w", err)
	}
	trx.ID = uint64(insertedID)
	transactionID := trx.ID

	if err := insertFinancialLedgerEntryTx(ctx, tx, &trx.TokoID, &transactionID, model.FinancialLedgerEntryWithdrawSettleDebit, trx.Amount, trx.Reference); err != nil {
		return err
	}
	if err := insertFinancialLedgerEntryTx(ctx, tx, &trx.TokoID, &transactionID, model.FinancialLedgerEntryWithdrawFeeDebit, fee, trx.Reference); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create pending withdraw transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *TransactionRepository) FinalizeWithdrawIfPending(ctx context.Context, id uint64, status model.TransactionStatus) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin finalize withdraw transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := `SELECT toko_id, amount, fee_withdrawal, reference, status, type
FROM transactions
WHERE id = ?
FOR UPDATE`
	var tokoID uint64
	var amount uint64
	var fee sql.NullInt64
	var reference sql.NullString
	var currentStatus model.TransactionStatus
	var trxType model.TransactionType
	if err := tx.QueryRowContext(ctx, query, id).Scan(&tokoID, &amount, &fee, &reference, &currentStatus, &trxType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, repository.ErrNotFound
		}
		return false, fmt.Errorf("load pending withdraw transaction: %w", err)
	}
	if trxType != model.TransactionTypeWithdraw {
		return false, repository.ErrNotFound
	}
	if currentStatus != model.TransactionStatusPending {
		return false, nil
	}

	updateQuery := `UPDATE transactions
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = ?`
	result, err := tx.ExecContext(ctx, updateQuery, status, id, model.TransactionStatusPending)
	if err != nil {
		return false, fmt.Errorf("update withdraw transaction status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected rows for withdraw finalize: %w", err)
	}
	if rowsAffected == 0 {
		return false, nil
	}

	if status == model.TransactionStatusFailed || status == model.TransactionStatusExpired {
		feeValue := uint64(0)
		if fee.Valid && fee.Int64 > 0 {
			feeValue = uint64(fee.Int64)
		}
		totalRefund, addErr := money.AddUint64(amount, feeValue)
		if addErr != nil {
			return false, fmt.Errorf("calculate withdraw refund total: %w", addErr)
		}
		delta := decimal.NewFromInt(int64(totalRefund)).StringFixed(2)
		refundQuery := `UPDATE balances
SET available = available + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ?`
		result, err = tx.ExecContext(ctx, refundQuery, delta, tokoID)
		if err != nil {
			return false, fmt.Errorf("refund withdraw settlement balance: %w", err)
		}
		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("read affected rows for withdraw refund: %w", err)
		}
		if rowsAffected == 0 {
			return false, repository.ErrNotFound
		}

		transactionID := id
		var referencePtr *string
		if reference.Valid {
			referencePtr = &reference.String
		}
		if err := insertFinancialLedgerEntryTx(ctx, tx, &tokoID, &transactionID, model.FinancialLedgerEntryWithdrawSettleRefund, amount, referencePtr); err != nil {
			return false, err
		}
		if err := insertFinancialLedgerEntryTx(ctx, tx, &tokoID, &transactionID, model.FinancialLedgerEntryWithdrawFeeRefund, feeValue, referencePtr); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit finalize withdraw transaction: %w", err)
	}
	committed = true
	return true, nil
}

func (r *TransactionRepository) ListPendingExpiryCandidates(ctx context.Context, olderThan time.Time, limit int) ([]repository.PendingExpiryCandidate, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
SELECT
  t.id,
  t.toko_id,
  tk.token,
  tk.callback_url,
  t.amount,
  COALESCE(t.reference, '') AS trx_id,
  '' AS rrn,
  COALESCE(t.code, '') AS custom_ref,
  'qris' AS vendor,
  t.created_at
FROM transactions t
INNER JOIN tokos tk ON tk.id = t.toko_id
WHERE t.status = ? AND t.type = ? AND t.created_at < ?
ORDER BY t.created_at ASC
LIMIT ?`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		model.TransactionStatusPending,
		model.TransactionTypeDeposit,
		olderThan.UTC(),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query pending expiry candidates: %w", err)
	}
	defer rows.Close()

	items := make([]repository.PendingExpiryCandidate, 0, limit)
	for rows.Next() {
		var item repository.PendingExpiryCandidate
		var callbackURL sql.NullString
		if err := rows.Scan(
			&item.TransactionID,
			&item.TokoID,
			&item.TokoToken,
			&callbackURL,
			&item.Amount,
			&item.TrxID,
			&item.RRN,
			&item.CustomRef,
			&item.Vendor,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending expiry candidate: %w", err)
		}
		if callbackURL.Valid {
			item.CallbackURL = &callbackURL.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending expiry candidates: %w", err)
	}

	return items, nil
}

func (r *TransactionRepository) GetDashboardMetricsByUser(ctx context.Context, userID uint64, from time.Time) (*repository.DashboardMetrics, error) {
	query := transactionVisibilityCTE() + `
SELECT
  COUNT(*) AS total_count,
  COALESCE(SUM(CASE WHEN t.status = 'success' THEN 1 ELSE 0 END), 0) AS success_count,
  COALESCE(SUM(CASE WHEN t.status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_count,
  COALESCE(SUM(CASE WHEN t.status IN ('failed', 'expired') THEN 1 ELSE 0 END), 0) AS failed_count,
  COALESCE(SUM(CASE WHEN t.status = 'success' AND t.type = 'deposit' THEN t.amount ELSE 0 END), 0) AS success_deposit_amount,
  COALESCE(SUM(CASE WHEN t.status = 'success' AND t.type = 'withdraw' THEN t.amount ELSE 0 END), 0) AS success_withdraw_amount,
  COALESCE(SUM(CASE WHEN t.status = 'success' THEN t.platform_fee ELSE 0 END), 0) AS total_platform_fee
FROM transactions t
INNER JOIN accessible_tokos at ON at.toko_id = t.toko_id
WHERE t.created_at >= ?
`

	metrics := &repository.DashboardMetrics{}
	if err := r.db.QueryRowContext(ctx, query, userID, from.UTC()).Scan(
		&metrics.TotalCount,
		&metrics.SuccessCount,
		&metrics.PendingCount,
		&metrics.FailedCount,
		&metrics.SuccessDepositAmount,
		&metrics.SuccessWithdrawAmount,
		&metrics.TotalPlatformFee,
	); err != nil {
		return nil, fmt.Errorf("query dashboard metrics: %w", err)
	}

	return metrics, nil
}

func (r *TransactionRepository) GetHourlyStatusCountsByUser(ctx context.Context, userID uint64, from time.Time) ([]repository.DashboardStatusSeriesPoint, error) {
	query := transactionVisibilityCTE() + `
SELECT
  DATE_FORMAT(t.created_at, '%Y-%m-%d %H:00:00') AS hour_bucket,
  COALESCE(SUM(CASE WHEN t.status = 'success' THEN 1 ELSE 0 END), 0) AS success_count,
  COALESCE(SUM(CASE WHEN t.status IN ('failed', 'expired') THEN 1 ELSE 0 END), 0) AS failed_count
FROM transactions t
INNER JOIN accessible_tokos at ON at.toko_id = t.toko_id
WHERE t.created_at >= ?
GROUP BY hour_bucket
ORDER BY hour_bucket ASC
`

	rows, err := r.db.QueryContext(ctx, query, userID, from.UTC())
	if err != nil {
		return nil, fmt.Errorf("query dashboard hourly volume: %w", err)
	}
	defer rows.Close()

	result := make([]repository.DashboardStatusSeriesPoint, 0)
	for rows.Next() {
		var hourBucket string
		var point repository.DashboardStatusSeriesPoint
		if err := rows.Scan(&hourBucket, &point.SuccessCount, &point.FailedCount); err != nil {
			return nil, fmt.Errorf("scan hourly status row: %w", err)
		}

		parsedBucket, err := time.ParseInLocation("2006-01-02 15:04:05", hourBucket, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("parse hourly status bucket: %w", err)
		}
		point.Bucket = parsedBucket

		result = append(result, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly status rows: %w", err)
	}

	return result, nil
}

func (r *TransactionRepository) ListRecentByUser(ctx context.Context, userID uint64, filter repository.TransactionHistoryFilter) ([]repository.TransactionHistoryRecord, error) {
	return r.listHistoryByUser(ctx, userID, filter, false)
}

func (r *TransactionRepository) ListRecentSuccessByUser(ctx context.Context, userID uint64, limit int) ([]repository.TransactionHistoryRecord, error) {
	return r.listHistoryByUser(ctx, userID, repository.TransactionHistoryFilter{
		Limit:  limit,
		Offset: 0,
	}, true)
}

func (r *TransactionRepository) CountHistoryByUser(ctx context.Context, userID uint64, filter repository.TransactionHistoryFilter) (uint64, error) {
	query := transactionVisibilityCTE() + `
SELECT COUNT(1)
FROM transactions t
INNER JOIN accessible_tokos at ON at.toko_id = t.toko_id
WHERE 1=1
`
	args := []any{userID}
	whereClause, whereArgs := buildHistoryFilterClause(filter, false)
	query += whereClause
	args = append(args, whereArgs...)

	var count uint64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count transaction history: %w", err)
	}
	return count, nil
}

func (r *TransactionRepository) listHistoryByUser(ctx context.Context, userID uint64, filter repository.TransactionHistoryFilter, successOnly bool) ([]repository.TransactionHistoryRecord, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := transactionVisibilityCTE() + `
SELECT
  t.id,
  t.toko_id,
  tk.name,
  t.player,
  t.code,
  t.type,
  t.status,
  t.reference,
  t.amount,
  t.netto,
  t.created_at
FROM transactions t
INNER JOIN accessible_tokos at ON at.toko_id = t.toko_id
INNER JOIN tokos tk ON tk.id = t.toko_id
WHERE 1=1
`
	args := []any{userID}
	whereClause, whereArgs := buildHistoryFilterClause(filter, successOnly)
	query += whereClause
	args = append(args, whereArgs...)
	query += `
ORDER BY t.created_at DESC
LIMIT ? OFFSET ?
`
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query transaction history: %w", err)
	}
	defer rows.Close()

	history := make([]repository.TransactionHistoryRecord, 0, filter.Limit)
	for rows.Next() {
		item := repository.TransactionHistoryRecord{}
		var player sql.NullString
		var code sql.NullString
		var reference sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.TokoID,
			&item.TokoName,
			&player,
			&code,
			&item.Type,
			&item.Status,
			&reference,
			&item.Amount,
			&item.Netto,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction history row: %w", err)
		}
		if player.Valid {
			item.Player = &player.String
		}
		if code.Valid {
			item.Code = &code.String
		}
		if reference.Valid {
			item.Reference = &reference.String
		}
		history = append(history, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transaction history rows: %w", err)
	}

	return history, nil
}

func buildHistoryFilterClause(filter repository.TransactionHistoryFilter, successOnly bool) (string, []any) {
	var builder strings.Builder
	args := make([]any, 0, 8)

	if successOnly {
		builder.WriteString(" AND t.status = ?")
		args = append(args, model.TransactionStatusSuccess)
	}
	if filter.From != nil {
		builder.WriteString(" AND t.created_at >= ?")
		args = append(args, filter.From.UTC())
	}
	if filter.To != nil {
		builder.WriteString(" AND t.created_at <= ?")
		args = append(args, filter.To.UTC())
	}
	if filter.Type != "" {
		builder.WriteString(" AND t.type = ?")
		args = append(args, filter.Type)
	}

	term := strings.TrimSpace(filter.SearchTerm)
	if term != "" {
		like := "%" + term + "%"
		builder.WriteString(" AND (t.reference LIKE ? OR t.player LIKE ? OR t.code LIKE ?)")
		args = append(args, like, like, like)
	}
	return builder.String(), args
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableUint64(value *uint64) any {
	if value == nil {
		return nil
	}
	return *value
}
