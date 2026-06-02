-- Phase 5g: Therapist Dashboard
-- therapists table + client-therapist opt-in links

CREATE TABLE therapists (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    email        TEXT        NOT NULL UNIQUE,
    credentials  TEXT,                          -- e.g. "PhD, Clinical Psychology"
    plan         TEXT        NOT NULL DEFAULT 'trial',  -- trial | active | cancelled
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_therapists_email ON therapists(email);

-- User explicitly opts in to share their journal summaries with a therapist.
CREATE TABLE client_therapist_links (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    therapist_id  UUID        NOT NULL REFERENCES therapists(id) ON DELETE CASCADE,
    client_id     UUID        NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    status        TEXT        NOT NULL DEFAULT 'active',  -- active | revoked
    linked_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at    TIMESTAMPTZ,
    UNIQUE (therapist_id, client_id)
);

CREATE INDEX idx_client_therapist_therapist ON client_therapist_links(therapist_id);
CREATE INDEX idx_client_therapist_client    ON client_therapist_links(client_id);
