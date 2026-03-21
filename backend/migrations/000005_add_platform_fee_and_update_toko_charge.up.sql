SET @platform_fee_col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND COLUMN_NAME = 'platform_fee'
);
SET @sql = IF(
  @platform_fee_col_exists = 0,
  "ALTER TABLE transactions ADD COLUMN platform_fee BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER fee_withdrawal",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

ALTER TABLE tokos
  MODIFY COLUMN charge INT NOT NULL DEFAULT 3;
