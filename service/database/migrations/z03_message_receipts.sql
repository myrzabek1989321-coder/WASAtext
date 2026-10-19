-- track delivery and reading of messages

CREATE TABLE message_receipts (
    message_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    delivered_at TEXT,
    read_at TEXT,

    PRIMARY KEY (message_id, user_id),

    FOREIGN KEY (message_id)
        REFERENCES messages(id)
        ON DELETE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES users(id),

    CHECK (read_at IS NULL OR delivered_at IS NOT NULL)
);

-- include messages created before this migration
INSERT INTO message_receipts (message_id, user_id)
SELECT m.id, cm.user_id
FROM messages m
JOIN conversation_members cm
    ON cm.conversation_id = m.conversation_id
WHERE cm.user_id != m.sender_id;
