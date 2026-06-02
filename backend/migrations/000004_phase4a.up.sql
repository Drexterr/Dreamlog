-- Phase 4a: Onboarding — goal selection and preferred name
ALTER TABLE users ADD COLUMN IF NOT EXISTS goal TEXT
    CHECK (goal IN ('stress', 'anxiety', 'grief', 'relationships', 'career', 'curious'));
ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_name TEXT;
