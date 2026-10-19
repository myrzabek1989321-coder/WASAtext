package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

func (db *appdbimpl) ForwardMessage(
	authID, sourceMessageID, targetConversationID string,
) (Message, error) {

	tx, err := db.c.Begin()
	if err != nil {
		return Message{}, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Find the authenticated user
	var sender User
	var senderMime sql.NullString
	var senderPhoto []byte

	err = tx.QueryRow(
		`SELECT id, name, photo_mime_type, photo_data
         FROM users WHERE id = ?`,
		authID,
	).Scan(
		&sender.ID, &sender.Name,
		&senderMime, &senderPhoto,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrUnknownUser
	}
	if err != nil {
		return Message{}, err
	}

	if senderMime.Valid && len(senderPhoto) > 0 {
		sender.Photo = &Photo{
			MimeType: senderMime.String,
			Data:     senderPhoto,
		}
	}

	// 2. Find the original message
	var sourceConversationID string
	var messageType string
	var originalText sql.NullString
	var originalMime sql.NullString
	var originalPhoto []byte

	err = tx.QueryRow(
		`SELECT conversation_id, type,
                text_content, photo_mime_type, photo_data
         FROM messages
         WHERE id = ?`,
		sourceMessageID,
	).Scan(
		&sourceConversationID,
		&messageType,
		&originalText,
		&originalMime,
		&originalPhoto,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrMessageNotFound
	}
	if err != nil {
		return Message{}, err
	}

	// 3. The sender must be a member of the source chat
	var exists int

	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversation_members
            WHERE conversation_id = ? AND user_id = ?
        )`,
		sourceConversationID, authID,
	).Scan(&exists)

	if err != nil {
		return Message{}, err
	}
	if exists == 0 {
		return Message{}, ErrConversationForbidden
	}

	// 4. The destination conversation must exist
	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversations WHERE id = ?
        )`,
		targetConversationID,
	).Scan(&exists)

	if err != nil {
		return Message{}, err
	}
	if exists == 0 {
		return Message{}, ErrConversationNotFound
	}

	// 5. The sender must belong to the destination chat
	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversation_members
            WHERE conversation_id = ? AND user_id = ?
        )`,
		targetConversationID, authID,
	).Scan(&exists)

	if err != nil {
		return Message{}, err
	}
	if exists == 0 {
		return Message{}, ErrConversationForbidden
	}

	// 6. Prepare the copied message content
	var textValue, mimeValue, photoValue any
	var textContent string
	var photo *Photo

	switch messageType {
	case "text":
		if !originalText.Valid {
			return Message{}, fmt.Errorf("missing original text")
		}
		textContent = originalText.String
		textValue = textContent

	case "photo":
		if !originalMime.Valid || len(originalPhoto) == 0 {
			return Message{}, fmt.Errorf("missing original photo")
		}
		mimeValue = originalMime.String
		photoValue = originalPhoto
		photo = &Photo{
			MimeType: originalMime.String,
			Data:     originalPhoto,
		}

	default:
		return Message{}, fmt.Errorf("unsupported message type")
	}

	// 7. Generate a new message ID and timestamp
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return Message{}, err
	}

	newMessageID := hex.EncodeToString(randomBytes)
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// 8. Insert the forwarded message
	_, err = tx.Exec(
		`INSERT INTO messages (
            id, conversation_id, sender_id, type,
            text_content, photo_mime_type, photo_data,
            forwarded_from_message_id, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newMessageID,
		targetConversationID,
		authID,
		messageType,
		textValue,
		mimeValue,
		photoValue,
		sourceMessageID,
		createdAt,
	)

	if err != nil {
		return Message{}, fmt.Errorf("insert forwarded message: %w", err)
	}

	// 9. Create delivery receipts for destination members
	_, err = tx.Exec(
		`INSERT INTO message_receipts (message_id, user_id)
         SELECT ?, user_id
         FROM conversation_members
         WHERE conversation_id = ? AND user_id != ?`,
		newMessageID, targetConversationID, authID,
	)

	if err != nil {
		return Message{}, fmt.Errorf("create receipts: %w", err)
	}

	// 10. Save all changes
	if err := tx.Commit(); err != nil {
		return Message{}, fmt.Errorf("commit forwarding: %w", err)
	}

	return Message{
		ID:                     newMessageID,
		ConversationID:         targetConversationID,
		Sender:                 sender,
		Type:                   messageType,
		Text:                   textContent,
		Photo:                  photo,
		CreatedAt:              createdAt,
		ForwardedFromMessageID: sourceMessageID,
		Reactions:              make([]any, 0),
		Delivered:              false,
		Read:                   false,
	}, nil
}
