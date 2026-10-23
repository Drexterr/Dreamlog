-- A follow-up conversation that triggers crisis screening flags its parent
-- entry. The entry's own text was screened safe, so its reflection stays
-- visible (is_crisis stays false), but - like a crisis entry - it no longer
-- feeds mood stats, reviews, team dashboards, insight cards, flashbacks,
-- therapist briefs or the context of future reflections and therapy sessions.
ALTER TABLE entry_analysis ADD COLUMN IF NOT EXISTS followup_crisis BOOLEAN NOT NULL DEFAULT false;

-- Entries whose follow-up already ended in crisis (conversations.is_crisis,
-- migration 000046).
UPDATE entry_analysis ea
SET followup_crisis = true
FROM conversations c
WHERE c.entry_id = ea.entry_id AND c.is_crisis;

CREATE OR REPLACE VIEW v_daily_mood AS
SELECT
    e.user_id,
    DATE(e.created_at AT TIME ZONE 'UTC') AS day,
    ROUND(AVG(ea.mood_score))::INT        AS avg_mood,
    COUNT(e.id)                           AS entry_count
FROM entries e
JOIN entry_analysis ea ON ea.entry_id = e.id
WHERE ea.mood_score IS NOT NULL
  AND NOT (ea.is_crisis OR ea.followup_crisis)
GROUP BY e.user_id, DATE(e.created_at AT TIME ZONE 'UTC');

CREATE OR REPLACE VIEW v_team_daily_mood AS
    SELECT
        cm.company_id,
        DATE(e.created_at AT TIME ZONE 'UTC') AS day,
        ROUND(AVG(ea.mood_score))::INT         AS avg_mood,
        COUNT(DISTINCT e.user_id)::INT         AS active_members,
        COUNT(e.id)::INT                       AS entry_count
    FROM company_members cm
    JOIN entries e        ON e.user_id  = cm.user_id  AND e.status = 'completed'
                         AND e.created_at >= cm.joined_at
    JOIN entry_analysis ea ON ea.entry_id = e.id      AND NOT (ea.is_crisis OR ea.followup_crisis)
    GROUP BY cm.company_id, DATE(e.created_at AT TIME ZONE 'UTC');
