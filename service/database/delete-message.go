package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrMessageNotFound = errors.New("message not found")
var ErrMessageForbidden = errors.New("cannot delete another user's message")

func (db *appdbimpl) DeleteMessage(authID, messageID string) error {

	tx, err := db.c.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Check that the authenticated user exists
	var userExists int

	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM users WHERE id = ?
        )`,
		authID,
	).Scan(&userExists)

	if err != nil {
		return fmt.Errorf("check user: %w", err)
	}

	if userExists == 0 {
		return ErrUnknownUser
	}

	// 2. Find the sender of the message
	var senderID string

	err = tx.QueryRow(
		`SELECT sender_id
         FROM messages
         WHERE id = ?`,
		messageID,
	).Scan(&senderID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrMessageNotFound
	}

	if err != nil {
		return fmt.Errorf("find message: %w", err)
	}

	// 3. A user can delete only their own messages
	if senderID != authID {
		return ErrMessageForbidden
	}

	// 4. Remove reply references to this message
	_, err = tx.Exec(
		`UPDATE messages
         SET reply_to_message_id = NULL
         WHERE reply_to_message_id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("clear reply references: %w", err)
	}

	// 5. Remove forwarding references
	_, err = tx.Exec(
		`UPDATE messages
         SET forwarded_from_message_id = NULL
         WHERE forwarded_from_message_id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("clear forwarding references: %w", err)
	}

	// 6. Delete reactions associated with this message
	_, err = tx.Exec(
		`DELETE FROM message_reactions
         WHERE message_id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("delete reactions: %w", err)
	}

	// 7. Delete delivery and read receipts
	_, err = tx.Exec(
		`DELETE FROM message_receipts
         WHERE message_id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("delete receipts: %w", err)
	}

	// 8. Finally, delete the message itself
	_, err = tx.Exec(
		`DELETE FROM messages
         WHERE id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	// 9. Save all changes
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit message deletion: %w", err)
	}

	return nil
}
