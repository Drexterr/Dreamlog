-- Phase 5c: B2B Corporate Wellness
-- companies table + company membership + aggregated team mood view

CREATE TABLE companies (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    slug         TEXT        NOT NULL UNIQUE,           -- used in API paths
    admin_email  TEXT        NOT NULL,
    plan         TEXT        NOT NULL DEFAULT 'trial',  -- trial | active | cancelled
    seat_limit   INT         NOT NULL DEFAULT 50,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_companies_slug ON companies(slug);

CREATE TABLE company_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT        NOT NULL DEFAULT 'member',  -- member | admin
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, user_id)
);

CREATE INDEX idx_company_members_company ON company_members(company_id);
CREATE INDEX idx_company_members_user    ON company_members(user_id);

-- View: anonymised per-day team mood for an HR dashboard.
-- Never exposes individual user identity - only aggregate stats per company.
CREATE VIEW v_team_daily_mood AS
    SELECT
        cm.company_id,
        DATE(e.created_at AT TIME ZONE 'UTC') AS day,
        ROUND(AVG(ea.mood_score))::INT         AS avg_mood,
        COUNT(DISTINCT e.user_id)::INT         AS active_members,
        COUNT(e.id)::INT                       AS entry_count
    FROM company_members cm
    JOIN entries e        ON e.user_id  = cm.user_id  AND e.status = 'completed'
    JOIN entry_analysis ea ON ea.entry_id = e.id      AND ea.is_crisis = FALSE
    GROUP BY cm.company_id, DATE(e.created_at AT TIME ZONE 'UTC');
