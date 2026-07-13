ALTER TABLE payments ALTER COLUMN store SET DEFAULT 'stripe';

ALTER TABLE payments RENAME COLUMN transaction_id TO payment_intent_id;
