-- Dream Decoder dual-lens interpretation fields.
-- psychological_lens: Jungian/depth-psychology reading of the dream symbols.
-- vedic_lens: Vedic Svapna Shastra / Hindu mythology reading of the dream symbols.
-- Both columns are NULL for non-dream entries (mode != 'dream').

ALTER TABLE entry_analysis
    ADD COLUMN IF NOT EXISTS psychological_lens TEXT,
    ADD COLUMN IF NOT EXISTS vedic_lens         TEXT;
