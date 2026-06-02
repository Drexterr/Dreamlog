ALTER TABLE therapy_sessions
    DROP COLUMN IF EXISTS crisis_warnings,
    DROP COLUMN IF EXISTS persona;
