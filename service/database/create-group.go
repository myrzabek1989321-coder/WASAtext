package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidGroupName = errors.New("invalid group name")
var ErrGroupMemberNotFound = errors.New("group member not found")

func (db *appdbimpl) CreateGroup(
	authID, name string,
	memberIDs []string,
) (ConversationSummary, error) {

	name = strings.TrimSpace(name)
	if name == "" {
		return ConversationSummary{}, ErrInvalidGroupName
	}

	tx, err := db.c.Begin()
	if err != nil {
		return ConversationSummary{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Check that the creator exists
	var exists int

	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return ConversationSummary{}, err
	}
	if exists == 0 {
		return ConversationSummary{}, ErrUnknownUser
	}

	// 2. Prepare a unique list of group members
	//  The creator is included automatically
	allMembers := []string{authID}
	seen := map[string]bool{authID: true}

	for _, userID := range memberIDs {
		if seen[userID] {
			continue
		}

		err = tx.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
			userID,
		).Scan(&exists)

		if err != nil {
			return ConversationSummary{}, err
		}
		if exists == 0 {
			return ConversationSummary{}, ErrGroupMemberNotFound
		}

		seen[userID] = true
		allMembers = append(allMembers, userID)
	}

	// 3. Generate unique conversation id
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return ConversationSummary{}, err
	}

	groupID := hex.EncodeToString(randomBytes)

	// 4. Create group conversation
	_, err = tx.Exec(
		`INSERT INTO conversations (id, type, name)
         VALUES (?, 'group', ?)`,
		groupID, name,
	)

	if err != nil {
		return ConversationSummary{}, fmt.Errorf("create group: %w", err)
	}

	// 5. Add the creator and all other members
	for _, userID := range allMembers {
		_, err = tx.Exec(
			`INSERT INTO conversation_members
             (conversation_id, user_id)
             VALUES (?, ?)`,
			groupID, userID,
		)

		if err != nil {
			return ConversationSummary{}, fmt.Errorf("add group member: %w", err)
		}
	}

	// 6. Read the group members and their photos
	rows, err := tx.Query(
		`SELECT u.id, u.name,
                u.photo_mime_type, u.photo_data
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

	group := ConversationSummary{
		ID:      groupID,
		Type:    "group",
		Name:    name,
		Members: make([]User, 0),
	}

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
			rows.Close()
			return ConversationSummary{}, fmt.Errorf("read group member: %w", err)
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
		rows.Close()
		return ConversationSummary{}, fmt.Errorf("iterate group members: %w", err)
	}

	if err := rows.Close(); err != nil {
		return ConversationSummary{}, fmt.Errorf("close group members: %w", err)
	}

	// 7. Save the group and its members
	if err := tx.Commit(); err != nil {
		return ConversationSummary{}, fmt.Errorf("commit group: %w", err)
	}

	return group, nil
}
