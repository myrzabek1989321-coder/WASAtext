// New File

package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrRecipientNotFound = errors.New("recipient not found")
var ErrInvalidRecipient = errors.New("invalid recipient")

type ConversationSummary struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Name               string `json:"name"`
	Photo              *Photo `json:"photo,omitempty"`
	Members            []User `json:"members"`
	LastMessageAt      string `json:"lastMessageAt,omitempty"`
	LastMessagePreview string `json:"lastMessagePreview,omitempty"`
	LastMessageType    string `json:"lastMessageType,omitempty"`
}

func (db *appdbimpl) CreateConversation(authID, recipientID string) (ConversationSummary, error) {
	var conversation ConversationSummary

	if authID == recipientID {
		return conversation, ErrInvalidRecipient
	}

	tx, err := db.c.Begin()
	if err != nil {
		return conversation, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Check that both users exist.
	var exists int

	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return conversation, err
	}
	if exists == 0 {
		return conversation, ErrUnknownUser
	}

	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		recipientID,
	).Scan(&exists)

	if err != nil {
		return conversation, err
	}
	if exists == 0 {
		return conversation, ErrRecipientNotFound
	}

	// keep user ids in the same order
	user1, user2 := authID, recipientID
	if user1 > user2 {
		user1, user2 = user2, user1
	}

	// check whether the private conversation already exists
	var conversationID string

	err = tx.QueryRow(
		`SELECT conversation_id
         FROM private_conversation_pairs
         WHERE user1_id = ? AND user2_id = ?`,
		user1, user2,
	).Scan(&conversationID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return conversation, fmt.Errorf("find conversation: %w", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		randomBytes := make([]byte, 16)
		if _, err := rand.Read(randomBytes); err != nil {
			return conversation, err
		}

		conversationID = hex.EncodeToString(randomBytes)

		_, err = tx.Exec(
			`INSERT INTO conversations (id, type)
             VALUES (?, 'private')`,
			conversationID,
		)
		if err != nil {
			return conversation, fmt.Errorf("create conversation: %w", err)
		}

		_, err = tx.Exec(
			`INSERT INTO private_conversation_pairs
             (conversation_id, user1_id, user2_id)
             VALUES (?, ?, ?)`,
			conversationID, user1, user2,
		)
		if err != nil {
			return conversation, fmt.Errorf("store private pair: %w", err)
		}

		for _, userID := range []string{user1, user2} {
			_, err = tx.Exec(
				`INSERT INTO conversation_members
                 (conversation_id, user_id)
                 VALUES (?, ?)`,
				conversationID, userID,
			)
			if err != nil {
				return conversation, fmt.Errorf("add member: %w", err)
			}
		}
	}

	// get the members and their profile photos
	rows, err := tx.Query(
		`SELECT u.id, u.name, u.photo_mime_type, u.photo_data
         FROM users u
         JOIN conversation_members cm ON cm.user_id = u.id
         WHERE cm.conversation_id = ?
         ORDER BY u.name COLLATE NOCASE`,
		conversationID,
	)
	if err != nil {
		return conversation, fmt.Errorf("get members: %w", err)
	}

	conversation = ConversationSummary{
		ID:      conversationID,
		Type:    "private",
		Members: make([]User, 0),
	}

	for rows.Next() {
		var user User
		var mimeType sql.NullString
		var photoData []byte

		err = rows.Scan(
			&user.ID, &user.Name, &mimeType, &photoData,
		)
		if err != nil {
			rows.Close()
			return ConversationSummary{}, err
		}

		if mimeType.Valid && len(photoData) > 0 {
			user.Photo = &Photo{
				MimeType: mimeType.String,
				Data:     photoData,
			}
		}

		conversation.Members = append(conversation.Members, user)

		if user.ID == recipientID {
			conversation.Name = user.Name
			conversation.Photo = user.Photo
		}
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return ConversationSummary{}, err
	}
	if err := rows.Close(); err != nil {
		return ConversationSummary{}, err
	}

	if err := tx.Commit(); err != nil {
		return ConversationSummary{}, fmt.Errorf("commit conversation: %w", err)
	}

	return conversation, nil
}
