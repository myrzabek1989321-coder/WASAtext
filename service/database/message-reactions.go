package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrReactionMessageNotFound = errors.New("message not found")
var ErrReactionForbidden = errors.New("not a conversation member")

type Reaction struct {
	User     User   `json:"user"`
	Emoticon string `json:"emoticon"`
}

// сheck that the user can access this message
func checkReactionAccess(
	tx *sql.Tx,
	authID, messageID string,
) error {
	var exists int

	err := tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrUnknownUser
	}

	var conversationID string

	err = tx.QueryRow(
		"SELECT conversation_id FROM messages WHERE id = ?",
		messageID,
	).Scan(&conversationID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrReactionMessageNotFound
	}
	if err != nil {
		return err
	}

	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversation_members
            WHERE conversation_id = ? AND user_id = ?
        )`,
		conversationID, authID,
	).Scan(&exists)

	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrReactionForbidden
	}

	return nil
}

// add or replace my reaction
func (db *appdbimpl) CommentMessage(
	authID, messageID, emoticon string,
) (Reaction, error) {

	tx, err := db.c.Begin()
	if err != nil {
		return Reaction{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := checkReactionAccess(tx, authID, messageID); err != nil {
		return Reaction{}, err
	}

	// read the user who is reacting
	var reaction Reaction
	var mimeType sql.NullString
	var photoData []byte

	err = tx.QueryRow(
		`SELECT id, name, photo_mime_type, photo_data
         FROM users WHERE id = ?`,
		authID,
	).Scan(
		&reaction.User.ID,
		&reaction.User.Name,
		&mimeType,
		&photoData,
	)
	if err != nil {
		return Reaction{}, fmt.Errorf("get reaction user: %w", err)
	}

	if mimeType.Valid && len(photoData) > 0 {
		reaction.User.Photo = &Photo{
			MimeType: mimeType.String,
			Data:     photoData,
		}
	}

	// remove my previous emoji on this message
	_, err = tx.Exec(
		`DELETE FROM message_reactions
         WHERE message_id = ? AND user_id = ?`,
		messageID, authID,
	)
	if err != nil {
		return Reaction{}, fmt.Errorf("replace reaction: %w", err)
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return Reaction{}, err
	}

	reactionID := hex.EncodeToString(randomBytes)

	_, err = tx.Exec(
		`INSERT INTO message_reactions
         (id, message_id, user_id, emoji)
         VALUES (?, ?, ?, ?)`,
		reactionID, messageID, authID, emoticon,
	)
	if err != nil {
		return Reaction{}, fmt.Errorf("save reaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Reaction{}, fmt.Errorf("commit reaction: %w", err)
	}

	reaction.Emoticon = emoticon
	return reaction, nil
}

// remove my reaction
func (db *appdbimpl) UncommentMessage(
	authID, messageID string,
) error {

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := checkReactionAccess(tx, authID, messageID); err != nil {
		return err
	}

	_, err = tx.Exec(
		`DELETE FROM message_reactions
         WHERE message_id = ? AND user_id = ?`,
		messageID, authID,
	)
	if err != nil {
		return fmt.Errorf("delete reaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reaction deletion: %w", err)
	}

	return nil
}
