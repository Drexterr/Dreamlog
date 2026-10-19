-- Account deletion is now a hard DELETE FROM users (every user-owned table
-- cascades). Payment rows must survive it: the unique transaction_id is the
-- replay guard that stops a store purchase granting a plan twice, and the
-- records are needed for financial reconciliation. Detach them instead.
ALTER TABLE payments ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_user_id_fkey;
ALTER TABLE payments
    ADD CONSTRAINT payments_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
