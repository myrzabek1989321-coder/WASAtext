CREATE TABLE message_reactions (
    id TEXT NOT NULL PRIMARY KEY,
    message_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    emoji TEXT NOT NULL,

    created_at TEXT NOT NULL
        DEFAULT (datetime('now')),

    UNIQUE (message_id, user_id, emoji),

    FOREIGN KEY (message_id)
        REFERENCES messages(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
);


CREATE INDEX idx_message_reactions_message
ON message_reactions(message_id);
