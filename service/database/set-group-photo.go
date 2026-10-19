package database

import (
	"database/sql"
	"errors"
	"fmt"
)

func (db *appdbimpl) SetGroupPhoto(
	authID, groupID string,
	photo Photo,
) (ConversationSummary, error) {

	tx, err := db.c.Begin()
	if err != nil {
		return ConversationSummary{}, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Check whether authenticated user exists
	var exists int

	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM users WHERE id = ?
        )`,
		authID,
	).Scan(&exists)

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("check user: %w", err)
	}

	if exists == 0 {
		return ConversationSummary{}, ErrUnknownUser
	}

	// Check that conversation exists and is a group
	var conversationType string

	err = tx.QueryRow(
		`SELECT type FROM conversations WHERE id = ?`,
		groupID,
	).Scan(&conversationType)

	if errors.Is(err, sql.ErrNoRows) {
		return ConversationSummary{}, ErrGroupNotFound
	}

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("find group: %w", err)
	}

	if conversationType != "group" {
		return ConversationSummary{}, ErrGroupNotFound
	}

	// Only current members can change group photo
	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversation_members
            WHERE conversation_id = ? AND user_id = ?
        )`,
		groupID, authID,
	).Scan(&exists)

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("check membership: %w", err)
	}

	if exists == 0 {
		return ConversationSummary{}, ErrGroupForbidden
	}

	// Save new image
	_, err = tx.Exec(
		`UPDATE conversations
         SET photo_mime_type = ?,
             photo_data = ?
         WHERE id = ? AND type = 'group'`,
		photo.MimeType,
		photo.Data,
		groupID,
	)

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("update group photo: %w", err)
	}

	// Get updated group data
	group, err := loadGroupSummary(tx, groupID)
	if err != nil {
		return ConversationSummary{}, err
	}

	// Commit all changes
	if err := tx.Commit(); err != nil {
		return ConversationSummary{}, fmt.Errorf("commit group photo: %w", err)
	}

	return group, nil
}
