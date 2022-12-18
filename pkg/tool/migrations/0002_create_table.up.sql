SET search_path TO music, public;

CREATE TABLE IF NOT EXISTS covers(
  id SERIAL NOT NULL PRIMARY KEY,

  artist TEXT NOT NULL,
  album TEXT NOT NULL,

  error_count INTEGER NOT NULL DEFAULT 0,

  completed BOOLEAN NOT NULL DEFAULT FALSE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX artist_album_idx ON covers(artist, album);
