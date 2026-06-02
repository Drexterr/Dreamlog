ALTER TABLE users ADD COLUMN age_range TEXT CHECK (age_range IN ('under_18', '18_24', '25_34', '35_44', '45_plus'));
