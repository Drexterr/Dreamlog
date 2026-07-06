-- Hook Model features: adaptive nudge timing + cross-entry connection insights.

-- When true, the morning nudge is scheduled at the user's learned typical
-- recording hour instead of the fixed fcm_nudge_hour. Explicitly setting
-- fcm_nudge_hour via PUT /me flips this off (manual choice wins).
ALTER TABLE users ADD COLUMN nudge_auto_time BOOLEAN NOT NULL DEFAULT true;

-- Users who already customized their nudge hour keep manual control.
UPDATE users SET nudge_auto_time = false WHERE fcm_nudge_hour <> 8;

-- Occasional cross-entry pattern insight ("this is the 3rd time X has come up
-- this month"), computed deterministically in the worker. NULL when no pattern
-- crossed the threshold - intentionally intermittent (variable reward).
ALTER TABLE entry_analysis ADD COLUMN connection_insight TEXT;
