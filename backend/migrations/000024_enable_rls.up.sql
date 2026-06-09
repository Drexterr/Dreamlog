-- Enable Row Level Security on all tables.
--
-- The backend connects to PostgreSQL directly via DATABASE_URL (superuser/service role),
-- which bypasses RLS. These statements only affect Supabase's PostgREST API, which this
-- app does not use - all data access goes through the Go backend.
--
-- Enabling RLS with no permissive policies = deny all PostgREST access, which is correct.
-- This clears the Supabase "RLS disabled" security advisory.

ALTER TABLE users                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE entries                ENABLE ROW LEVEL SECURITY;
ALTER TABLE entry_analysis         ENABLE ROW LEVEL SECURITY;
ALTER TABLE emotional_metrics      ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversations          ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversation_messages  ENABLE ROW LEVEL SECURITY;
ALTER TABLE dead_letter_jobs       ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_devices           ENABLE ROW LEVEL SECURITY;
ALTER TABLE nudges                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE weekly_reviews         ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_reviews         ENABLE ROW LEVEL SECURITY;
ALTER TABLE share_links            ENABLE ROW LEVEL SECURITY;
ALTER TABLE insight_shares         ENABLE ROW LEVEL SECURITY;
ALTER TABLE life_chapters          ENABLE ROW LEVEL SECURITY;
ALTER TABLE people                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE person_mentions        ENABLE ROW LEVEL SECURITY;
ALTER TABLE therapy_sessions       ENABLE ROW LEVEL SECURITY;
ALTER TABLE therapy_session_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE journey_sessions       ENABLE ROW LEVEL SECURITY;
ALTER TABLE journey_session_steps  ENABLE ROW LEVEL SECURITY;
ALTER TABLE streak_freeze_days     ENABLE ROW LEVEL SECURITY;
ALTER TABLE companies              ENABLE ROW LEVEL SECURITY;
ALTER TABLE company_members        ENABLE ROW LEVEL SECURITY;
ALTER TABLE therapists             ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_therapist_links ENABLE ROW LEVEL SECURITY;
