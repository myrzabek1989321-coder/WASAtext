
-- conversations (private chats and groups)
CREATE TABLE conversations (
    id TEXT NOT NULL PRIMARY KEY,
    type TEXT NOT NULL
        CHECK (type IN ('private', 'group')),
    name TEXT,
    created_at TEXT NOT NULL
        DEFAULT (datetime('now'))
);


-- users who belong to each conversation
CREATE TABLE conversation_members (
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    last_read_at TEXT,

    PRIMARY KEY (conversation_id, user_id),

    FOREIGN KEY (conversation_id)
        REFERENCES conversations(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id)
);



-- prevent duplicate private conversations
CREATE TABLE private_conversation_pairs (
    conversation_id TEXT NOT NULL PRIMARY KEY,
    user1_id TEXT NOT NULL,
    user2_id TEXT NOT NULL,

    UNIQUE (user1_id, user2_id),
    CHECK (user1_id < user2_id),

    FOREIGN KEY (conversation_id)
        REFERENCES conversations(id)
        ON DELETE CASCADE
);
