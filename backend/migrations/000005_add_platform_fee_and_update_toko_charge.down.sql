ALTER TABLE tokos
  MODIFY COLUMN charge INT NOT NULL DEFAULT 2;

SET @platform_fee_col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'transactions'
    AND COLUMN_NAME = 'platform_fee'
);
SET @sql = IF(
  @platform_fee_col_exists > 0,
  "ALTER TABLE transactions DROP COLUMN platform_fee",
  "SELECT 1"
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
