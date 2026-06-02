-- Relationship Map: person extraction + mention tracking.

CREATE TABLE people (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              TEXT        NOT NULL,
    role              TEXT        NOT NULL DEFAULT 'other'
                                  CHECK (role IN ('family', 'friend', 'colleague', 'romantic', 'other')),
    mention_count     INT         NOT NULL DEFAULT 1,
    positive_count    INT         NOT NULL DEFAULT 0,
    negative_count    INT         NOT NULL DEFAULT 0,
    last_mentioned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Case-insensitive uniqueness: "sarah" and "Sarah" are the same person.
CREATE UNIQUE INDEX idx_people_user_name ON people (user_id, LOWER(name));
CREATE INDEX idx_people_user_id ON people (user_id, last_mentioned_at DESC);

CREATE TABLE person_mentions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id  UUID        NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    entry_id   UUID        NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sentiment  TEXT        NOT NULL DEFAULT 'neutral'
                           CHECK (sentiment IN ('positive', 'neutral', 'negative')),
    context    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (person_id, entry_id)
);

CREATE INDEX idx_person_mentions_person ON person_mentions (person_id, created_at DESC);
