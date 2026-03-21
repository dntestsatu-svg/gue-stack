SET @idx_status_created_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND INDEX_NAME = 'idx_transactions_status_created_at'
);
SET @sql = IF(
  @idx_status_created_exists = 0,
  "ALTER TABLE transactions ADD INDEX idx_transactions_status_created_at (status, created_at)",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @uniq_toko_reference_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND INDEX_NAME = 'uniq_transactions_toko_reference'
);

SET @dup_toko_reference_count = (
  SELECT COUNT(*)
  FROM (
    SELECT toko_id, reference
    FROM transactions
    WHERE reference IS NOT NULL
    GROUP BY toko_id, reference
    HAVING COUNT(*) > 1
  ) d
);

SET @sql = IF(
  @uniq_toko_reference_exists = 0 AND @dup_toko_reference_count = 0,
  "ALTER TABLE transactions ADD UNIQUE KEY uniq_transactions_toko_reference (toko_id, reference)",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
