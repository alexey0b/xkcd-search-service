CREATE TABLE IF NOT EXISTS comics (
    id BIGINT PRIMARY KEY,
    url TEXT NOT NULL,
    words TEXT[]
);

CREATE TABLE IF NOT EXISTS comics_stats (
    comics_fetched BIGINT,
    words_total BIGINT,
    words_unique BIGINT
);

INSERT INTO comics_stats (comics_fetched, words_total, words_unique)
VALUES (0, 0, 0);
