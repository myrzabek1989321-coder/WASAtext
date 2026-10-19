// New file

package database

import (
	"database/sql"
	"fmt"
)

func (db *appdbimpl) GetMyConversations(authID string) ([]ConversationSummary, error) {
	// Check if the user exists.
	var exists int
	err := db.c.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		authID,
	).Scan(&exists)

	if err != nil {
		return nil, fmt.Errorf("check user: %w", err)
	}
	if exists == 0 {
		return nil, ErrUnknownUser
	}

	// find conversations where this user is a member
	rows, err := db.c.Query(
		`SELECT c.id, c.type, COALESCE(c.name, '')
         FROM conversations c
         JOIN conversation_members cm
             ON cm.conversation_id = c.id
         WHERE cm.user_id = ?
         ORDER BY c.created_at DESC, c.id DESC`,
		authID,
	)
	if err != nil {
		return nil, fmt.Errorf("get conversations: %w", err)
	}

	conversations := make([]ConversationSummary, 0)

	for rows.Next() {
		var conversation ConversationSummary

		err := rows.Scan(
			&conversation.ID,
			&conversation.Type,
			&conversation.Name,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("read conversation: %w", err)
		}

		conversations = append(conversations, conversation)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close conversations: %w", err)
	}

	// load members and their profile photos
	for i := range conversations {
		memberRows, err := db.c.Query(
			`SELECT u.id, u.name, u.photo_mime_type, u.photo_data
             FROM users u
             JOIN conversation_members cm
                 ON cm.user_id = u.id
             WHERE cm.conversation_id = ?
             ORDER BY u.name COLLATE NOCASE`,
			conversations[i].ID,
		)
		if err != nil {
			return nil, fmt.Errorf("get members: %w", err)
		}

		conversations[i].Members = make([]User, 0)

		for memberRows.Next() {
			var user User
			var mimeType sql.NullString
			var photoData []byte

			err := memberRows.Scan(
				&user.ID,
				&user.Name,
				&mimeType,
				&photoData,
			)
			if err != nil {
				memberRows.Close()
				return nil, fmt.Errorf("read member: %w", err)
			}

			if mimeType.Valid && len(photoData) > 0 {
				user.Photo = &Photo{
					MimeType: mimeType.String,
					Data:     photoData,
				}
			}

			conversations[i].Members = append(
				conversations[i].Members, user,
			)

			// Private chat shows the other user's name and photo.
			if conversations[i].Type == "private" && user.ID != authID {
				conversations[i].Name = user.Name
				conversations[i].Photo = user.Photo
			}
		}

		if err := memberRows.Err(); err != nil {
			memberRows.Close()
			return nil, fmt.Errorf("iterate members: %w", err)
		}
		if err := memberRows.Close(); err != nil {
			return nil, fmt.Errorf("close members: %w", err)
		}
	}

	return conversations, nil
}
