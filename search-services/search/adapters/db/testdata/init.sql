CREATE TABLE IF NOT EXISTS comics (
    id BIGINT PRIMARY KEY,
    url TEXT NOT NULL,
    words TEXT[]
);
