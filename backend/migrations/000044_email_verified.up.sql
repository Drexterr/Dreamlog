-- N6: an email is only proof of identity once a trusted provider verified it.
-- Local /auth/register accepts any address unverified; Supabase sign-ins are
-- verified (confirmed email or OAuth). Email-based authorization (B2B domain
-- join, sandbox purchase allowlist) requires email_verified.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;

UPDATE users SET email_verified = true WHERE supabase_id NOT LIKE 'local-%';
