UPDATE transactions
SET status = 'failed'
WHERE status = 'expired';

ALTER TABLE transactions
  MODIFY COLUMN status ENUM('pending', 'success', 'failed') NOT NULL;
