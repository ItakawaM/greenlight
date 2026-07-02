CREATE TABLE IF NOT EXISTS movies (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title text NOT NULL,
    year integer NOT NULL,
    runtime integer NOT NULL,
    genres text [] NOT NULL,
    version integer NOT NULL DEFAULT 1
);

ALTER TABLE movies
ADD CONSTRAINT movies_runtime_check CHECK (runtime > 0);

ALTER TABLE movies
ADD CONSTRAINT movies_year_check CHECK (
    year BETWEEN 1888 AND date_part('year', now())
);

ALTER TABLE movies
ADD CONSTRAINT movies_genres_length_check CHECK (
    cardinality(genres) BETWEEN 1 AND 5
);