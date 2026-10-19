ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_user_id_fkey;
DELETE FROM payments WHERE user_id IS NULL;
ALTER TABLE payments ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE payments
    ADD CONSTRAINT payments_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
