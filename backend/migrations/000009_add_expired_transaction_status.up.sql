ALTER TABLE transactions
  MODIFY COLUMN status ENUM('pending', 'success', 'failed', 'expired') NOT NULL;
