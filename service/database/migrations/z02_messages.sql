
-- store text and photo messages
CREATE TABLE messages (
    id TEXT NOT NULL PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('text', 'photo')),

    text_content TEXT,
    photo_mime_type TEXT,
    photo_data BLOB,

    reply_to_message_id TEXT,
    forwarded_from_message_id TEXT,
    created_at TEXT NOT NULL,

    FOREIGN KEY (conversation_id)
        REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id)
        REFERENCES users(id),
    FOREIGN KEY (reply_to_message_id)
        REFERENCES messages(id),

    CHECK (
        (type = 'text' AND text_content IS NOT NULL
            AND length(text_content) > 0 AND photo_data IS NULL)
        OR
        (type = 'photo' AND photo_data IS NOT NULL
            AND length(photo_data) > 0 AND photo_mime_type IS NOT NULL
            AND text_content IS NULL)
    )
);

CREATE INDEX idx_messages_conversation
ON messages(conversation_id, created_at DESC, id DESC);
