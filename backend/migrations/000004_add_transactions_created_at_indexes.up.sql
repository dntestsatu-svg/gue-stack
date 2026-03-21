SET @idx_tx_toko_created_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND INDEX_NAME = 'idx_transactions_toko_created_at'
);
SET @sql = IF(
  @idx_tx_toko_created_exists = 0,
  "ALTER TABLE transactions ADD INDEX idx_transactions_toko_created_at (toko_id, created_at)",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_tx_created_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND INDEX_NAME = 'idx_transactions_created_at'
);
SET @sql = IF(
  @idx_tx_created_exists = 0,
  "ALTER TABLE transactions ADD INDEX idx_transactions_created_at (created_at)",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
