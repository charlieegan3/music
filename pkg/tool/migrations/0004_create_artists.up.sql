SET search_path TO music, public;

CREATE TABLE IF NOT EXISTS artists(
    id NUMERIC PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(name)
);
