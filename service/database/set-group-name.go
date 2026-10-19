package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrGroupNotFound = errors.New("group not found")
var ErrGroupForbidden = errors.New("not a group member")

func (db *appdbimpl) SetGroupName(
	authID, groupID, name string,
) (ConversationSummary, error) {

	name = strings.TrimSpace(name)
	if name == "" {
		return ConversationSummary{}, ErrInvalidGroupName
	}

	tx, err := db.c.Begin()
	if err != nil {
		return ConversationSummary{}, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// Verify authenticated user
	var exists int

	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("check user: %w", err)
	}
	if exists == 0 {
		return ConversationSummary{}, ErrUnknownUser
	}

	// Verify this conversation is a group
	var conversationType string

	err = tx.QueryRow(
		"SELECT type FROM conversations WHERE id = ?",
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

	// Only current members can change group name
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

	// Save new group name
	_, err = tx.Exec(
		`UPDATE conversations
         SET name = ?
         WHERE id = ? AND type = 'group'`,
		name, groupID,
	)

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("update group name: %w", err)
	}

	// Load the updated group without modifying read receipts
	group, err := loadGroupSummary(tx, groupID)
	if err != nil {
		return ConversationSummary{}, err
	}

	if err := tx.Commit(); err != nil {
		return ConversationSummary{}, fmt.Errorf("commit group name: %w", err)
	}

	return group, nil
}

// Read group information inside same transaction
func loadGroupSummary(
	tx *sql.Tx,
	groupID string,
) (ConversationSummary, error) {

	var group ConversationSummary
	// Group photo
	var groupMime sql.NullString
	var groupPhoto []byte

	err := tx.QueryRow(
		`SELECT
            c.id,
            c.type,
            COALESCE(c.name, ''),
			c.photo_mime_type,
			c.photo_data,
            COALESCE(m.created_at, ''),
            COALESCE(m.type, ''),
            CASE
                WHEN m.type = 'photo' THEN '[Photo]'
                ELSE COALESCE(m.text_content, '')
            END
         FROM conversations c
         LEFT JOIN messages m ON m.id = (
             SELECT id
             FROM messages
             WHERE conversation_id = c.id
             ORDER BY created_at DESC, id DESC
             LIMIT 1
         )
         WHERE c.id = ? AND c.type = 'group'`,
		groupID,
	).Scan(
		&group.ID,
		&group.Type,
		&group.Name,
		&groupMime,
		&groupPhoto,
		&group.LastMessageAt,
		&group.LastMessageType,
		&group.LastMessagePreview,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ConversationSummary{}, ErrGroupNotFound
	}
	if err != nil {
		return ConversationSummary{}, fmt.Errorf("read group: %w", err)
	}

	if groupMime.Valid && len(groupPhoto) > 0 {
		group.Photo = &Photo{
			MimeType: groupMime.String,
			Data:     groupPhoto,
		}
	}

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
		return ConversationSummary{}, fmt.Errorf("get group members: %w", err)
	}

	group.Members = make([]User, 0)

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
			return ConversationSummary{}, fmt.Errorf("read member: %w", err)
		}

		if mimeType.Valid && len(photoData) > 0 {
			user.Photo = &Photo{
				MimeType: mimeType.String,
				Data:     photoData,
			}
		}

		group.Members = append(group.Members, user)
	}

	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return ConversationSummary{}, fmt.Errorf("iterate members: %w", err)
	}

	if err := rows.Close(); err != nil {
		return ConversationSummary{}, fmt.Errorf("close members: %w", err)
	}

	return group, nil
}
