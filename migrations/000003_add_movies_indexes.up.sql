CREATE INDEX IF NOT EXISTS movies_title_idx on movies USING GIN (to_tsvector('simple', title));

CREATE INDEX IF NOT EXISTS movies_genres_idx on movies USING GIN (genres);