// New file

package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var ErrConversationNotFound = errors.New("conversation not found")
var ErrConversationForbidden = errors.New("not a conversation member")
var ErrInvalidReply = errors.New("invalid reply message")

type SendMessageInput struct {
	Type             string
	Text             string
	Photo            *Photo
	ReplyToMessageID string
}

type Message struct {
	ID                     string `json:"id"`
	ConversationID         string `json:"conversationId"`
	Sender                 User   `json:"sender"`
	Type                   string `json:"type"`
	Text                   string `json:"text,omitempty"`
	Photo                  *Photo `json:"photo,omitempty"`
	CreatedAt              string `json:"createdAt"`
	ReplyToMessageID       string `json:"replyToMessageId,omitempty"`
	ForwardedFromMessageID string `json:"forwardedFromMessageId,omitempty"`
	Reactions              []any  `json:"reactions"`
	Delivered              bool   `json:"delivered"`
	Read                   bool   `json:"read"`
}

func (db *appdbimpl) SendMessage(
	authID, conversationID string,
	input SendMessageInput,
) (Message, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback()

	// find sender
	var sender User
	var mime sql.NullString
	var photoData []byte

	err = tx.QueryRow(
		`SELECT name, photo_mime_type, photo_data
         FROM users WHERE id = ?`, authID,
	).Scan(&sender.Name, &mime, &photoData)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrUnknownUser
	}
	if err != nil {
		return Message{}, err
	}

	sender.ID = authID
	if mime.Valid && len(photoData) > 0 {
		sender.Photo = &Photo{
			MimeType: mime.String,
			Data:     photoData,
		}
	}

	// check that conversation exists
	var exists int
	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM conversations WHERE id = ?)",
		conversationID,
	).Scan(&exists)
	if err != nil {
		return Message{}, err
	}
	if exists == 0 {
		return Message{}, ErrConversationNotFound
	}

	// only conversation members can send messages
	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversation_members
            WHERE conversation_id = ? AND user_id = ?
        )`,
		conversationID, authID,
	).Scan(&exists)
	if err != nil {
		return Message{}, err
	}
	if exists == 0 {
		return Message{}, ErrConversationForbidden
	}

	// reply must reference a message from same chat
	if input.ReplyToMessageID != "" {
		err = tx.QueryRow(
			`SELECT EXISTS(
                SELECT 1 FROM messages
                WHERE id = ? AND conversation_id = ?
            )`,
			input.ReplyToMessageID, conversationID,
		).Scan(&exists)
		if err != nil {
			return Message{}, err
		}
		if exists == 0 {
			return Message{}, ErrInvalidReply
		}
	}

	randomBytes := make([]byte, 16)
	if _, err = rand.Read(randomBytes); err != nil {
		return Message{}, err
	}

	messageID := hex.EncodeToString(randomBytes)
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	var textValue, mimeValue, imageValue, replyValue any

	if input.Type == "text" {
		textValue = input.Text
	} else if input.Type == "photo" && input.Photo != nil {
		mimeValue = input.Photo.MimeType
		imageValue = input.Photo.Data
	} else {
		return Message{}, fmt.Errorf("invalid message type")
	}

	if input.ReplyToMessageID != "" {
		replyValue = input.ReplyToMessageID
	}

	_, err = tx.Exec(
		`INSERT INTO messages
         (id, conversation_id, sender_id, type,
          text_content, photo_mime_type, photo_data,
          reply_to_message_id, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		messageID, conversationID, authID, input.Type,
		textValue, mimeValue, imageValue,
		replyValue, createdAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf("save message: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return Message{}, err
	}

	return Message{
		ID:               messageID,
		ConversationID:   conversationID,
		Sender:           sender,
		Type:             input.Type,
		Text:             input.Text,
		Photo:            input.Photo,
		CreatedAt:        createdAt,
		ReplyToMessageID: input.ReplyToMessageID,
		Reactions:        make([]any, 0),
		Delivered:        false,
		Read:             false,
	}, nil
}
