DROP TRIGGER IF EXISTS trg_entry_analysis_updated_at ON entry_analysis;
DROP TRIGGER IF EXISTS trg_entries_updated_at ON entries;
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP FUNCTION IF EXISTS trigger_set_updated_at();

DROP TABLE IF EXISTS dead_letter_jobs;
DROP TABLE IF EXISTS emotional_metrics;
DROP TABLE IF EXISTS entry_analysis;
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS entry_status;
