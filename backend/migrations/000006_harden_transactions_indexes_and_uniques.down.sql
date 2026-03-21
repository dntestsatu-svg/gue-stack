SET @uniq_toko_reference_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND INDEX_NAME = 'uniq_transactions_toko_reference'
);
SET @sql = IF(
  @uniq_toko_reference_exists > 0,
  "ALTER TABLE transactions DROP INDEX uniq_transactions_toko_reference",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_status_created_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND INDEX_NAME = 'idx_transactions_status_created_at'
);
SET @sql = IF(
  @idx_status_created_exists > 0,
  "ALTER TABLE transactions DROP INDEX idx_transactions_status_created_at",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
