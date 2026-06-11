-- Extend payments table for IAP / RevenueCat integration.
-- store: which store the purchase came from (stripe | apple | google | manual)
-- product_id: the SKU as known to the store (e.g. com.dreamlog.app.plus_monthly)
-- country: two-letter ISO country code from the store receipt (for FX tracking)

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS store      TEXT NOT NULL DEFAULT 'stripe',
    ADD COLUMN IF NOT EXISTS product_id TEXT,
    ADD COLUMN IF NOT EXISTS country    CHAR(2);
