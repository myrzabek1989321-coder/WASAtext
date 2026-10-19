package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrAddGroupNotFound = errors.New("group not found")
var ErrAddGroupForbidden = errors.New("not a group member")
var ErrAddGroupUserNotFound = errors.New("user to add not found")

func (db *appdbimpl) AddToGroup(
	authID, groupID string,
	memberIDs []string,
) ([]User, error) {

	tx, err := db.c.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Check the authenticated user
	var exists int

	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("check user: %w", err)
	}

	if exists == 0 {
		return nil, ErrUnknownUser
	}

	// Check that the conversation is a group
	var conversationType string

	err = tx.QueryRow(
		"SELECT type FROM conversations WHERE id = ?",
		groupID,
	).Scan(&conversationType)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAddGroupNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find group: %w", err)
	}

	if conversationType != "group" {
		return nil, ErrAddGroupNotFound
	}

	// Only current group members can add people
	err = tx.QueryRow(
		`SELECT EXISTS(
            SELECT 1 FROM conversation_members
            WHERE conversation_id = ? AND user_id = ?
        )`,
		groupID, authID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}

	if exists == 0 {
		return nil, ErrAddGroupForbidden
	}

	// Validate every user before adding anyone
	for _, userID := range memberIDs {
		err = tx.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
			userID,
		).Scan(&exists)

		if err != nil {
			return nil, fmt.Errorf("check new member: %w", err)
		}

		if exists == 0 {
			return nil, ErrAddGroupUserNotFound
		}
	}

	// Add new members without duplicating existing ones
	for _, userID := range memberIDs {
		_, err = tx.Exec(
			`INSERT INTO conversation_members
             (conversation_id, user_id)
             VALUES (?, ?)
             ON CONFLICT(conversation_id, user_id)
             DO NOTHING`,
			groupID, userID,
		)

		if err != nil {
			return nil, fmt.Errorf("add member: %w", err)
		}
	}

	// Return the updated list of group members
	rows, err := tx.Query(
		`SELECT
            u.id,
            u.name,
            u.photo_mime_type,
            u.photo_data
         FROM users u
         JOIN conversation_members cm
             ON cm.user_id = u.id
         WHERE cm.conversation_id = ?
         ORDER BY u.name COLLATE NOCASE`,
		groupID,
	)

	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	members := make([]User, 0)

	for rows.Next() {
		var user User
		var mimeType sql.NullString
		var photoData []byte

		err = rows.Scan(
			&user.ID,
			&user.Name,
			&mimeType,
			&photoData,
		)

		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read member: %w", err)
		}

		if mimeType.Valid && len(photoData) > 0 {
			user.Photo = &Photo{
				MimeType: mimeType.String,
				Data:     photoData,
			}
		}

		members = append(members, user)
	}

	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate members: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close members: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit group members: %w", err)
	}

	return members, nil
}
