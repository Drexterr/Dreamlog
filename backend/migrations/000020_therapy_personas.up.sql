-- Phase 8: Enhanced Therapy Mode - persona + layered crisis tracking

ALTER TABLE therapy_sessions
    ADD COLUMN persona         TEXT NOT NULL DEFAULT 'comforting'
        CHECK (persona IN ('comforting', 'rational', 'cbt', 'mindful')),
    ADD COLUMN crisis_warnings INT  NOT NULL DEFAULT 0;
