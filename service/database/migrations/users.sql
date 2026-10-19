

CREATE TABLE users (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
        CHECK (length(name) BETWEEN 3 AND 16)
);
