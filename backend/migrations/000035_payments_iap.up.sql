-- Payments move from Stripe PaymentIntents to store In-App Purchases.
-- The unique replay-protection column now holds a store transaction ID
-- (Apple transaction_id / Play orderId), so rename it to match reality.
-- Existing Stripe rows keep their intent IDs as historical transaction IDs.

ALTER TABLE payments RENAME COLUMN payment_intent_id TO transaction_id;

-- 'stripe' is no longer the norm; require the store to be set explicitly.
ALTER TABLE payments ALTER COLUMN store DROP DEFAULT;
