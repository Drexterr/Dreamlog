CREATE TABLE launch_waitlist (
    email TEXT PRIMARY KEY CHECK (length(email) <= 254 AND email = lower(email)),
    consent_version TEXT NOT NULL DEFAULT 'launch-notification-v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- This is an API-only table: never expose email addresses through Supabase REST.
ALTER TABLE launch_waitlist ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON launch_waitlist FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON launch_waitlist FROM authenticated;
    END IF;
END $$;
