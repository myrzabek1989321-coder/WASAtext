package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrLeaveGroupNotFound = errors.New("group not found")
var ErrLeaveGroupForbidden = errors.New("user is not a group member")

func (db *appdbimpl) LeaveGroup(authID, groupID string) error {
	tx, err := db.c.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Check that the authenticated user exists
	var exists int

	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return fmt.Errorf("check user: %w", err)
	}

	if exists == 0 {
		return ErrUnknownUser
	}

	// Check that conversation exists and is group
	var conversationType string

	err = tx.QueryRow(
		"SELECT type FROM conversations WHERE id = ?",
		groupID,
	).Scan(&conversationType)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrLeaveGroupNotFound
	}

	if err != nil {
		return fmt.Errorf("find group: %w", err)
	}

	if conversationType != "group" {
		return ErrLeaveGroupNotFound
	}

	// Remove the user from the group
	result, err := tx.Exec(
		`DELETE FROM conversation_members
         WHERE conversation_id = ? AND user_id = ?`,
		groupID, authID,
	)

	if err != nil {
		return fmt.Errorf("remove group member: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted rows: %w", err)
	}

	if affected == 0 {
		return ErrLeaveGroupForbidden
	}

	// Remove this users receipts from the group
	// Remaining members determine delivery/read status
	_, err = tx.Exec(
		`DELETE FROM message_receipts
         WHERE user_id = ?
           AND message_id IN (
               SELECT id FROM messages
               WHERE conversation_id = ?
           )`,
		authID, groupID,
	)

	if err != nil {
		return fmt.Errorf("remove group receipts: %w", err)
	}

	// Save changes together
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit group leave: %w", err)
	}

	return nil
}
