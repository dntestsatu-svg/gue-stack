CREATE TABLE IF NOT EXISTS financial_ledger_entries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  toko_id BIGINT UNSIGNED NULL,
  transaction_id BIGINT UNSIGNED NULL,
  entry_type VARCHAR(64) NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  reference VARCHAR(255) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_financial_ledger_toko_id (toko_id),
  KEY idx_financial_ledger_transaction_id (transaction_id),
  KEY idx_financial_ledger_entry_type (entry_type),
  KEY idx_financial_ledger_reference (reference),
  UNIQUE KEY uniq_financial_ledger_tx_type (transaction_id, entry_type),
  CONSTRAINT fk_financial_ledger_toko FOREIGN KEY (toko_id) REFERENCES tokos(id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT fk_financial_ledger_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
SELECT t.toko_id, t.id, 'deposit_pending_credit', t.netto, t.reference
FROM transactions t
WHERE t.type = 'deposit' AND t.status = 'success' AND t.netto > 0
  AND NOT EXISTS (
    SELECT 1
    FROM financial_ledger_entries fle
    WHERE fle.transaction_id = t.id AND fle.entry_type = 'deposit_pending_credit'
  );

INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
SELECT t.toko_id, t.id, 'project_platform_fee_credit', t.platform_fee, t.reference
FROM transactions t
WHERE t.type = 'deposit' AND t.status = 'success' AND t.platform_fee > 0
  AND NOT EXISTS (
    SELECT 1
    FROM financial_ledger_entries fle
    WHERE fle.transaction_id = t.id AND fle.entry_type = 'project_platform_fee_credit'
  );
