CREATE OR REPLACE VIEW v_team_daily_mood AS
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
