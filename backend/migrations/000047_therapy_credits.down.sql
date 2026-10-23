-- Reverses 000047. Therapy credit purchases cannot be represented once the
-- 'therapy' payment kind is gone, so their payment rows are removed with it.
DELETE FROM payments WHERE plan = 'therapy';
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_plan_check;
ALTER TABLE payments
    ADD CONSTRAINT payments_plan_check CHECK (plan IN ('plus', 'pro'));

DROP TABLE IF EXISTS therapy_credit_ledger;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_therapy_credits_nonnegative;
ALTER TABLE users DROP COLUMN IF EXISTS therapy_credits;
