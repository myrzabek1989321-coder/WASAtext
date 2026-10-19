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

	_, err = db.c.Exec(
		`UPDATE message_receipts
	     SET delivered_at = datetime('now')
	     WHERE user_id = ?
	       AND delivered_at IS NULL
	       AND EXISTS (
	           SELECT 1
	           FROM messages m
	           JOIN conversation_members cm
	             ON cm.conversation_id = m.conversation_id
	           WHERE m.id = message_receipts.message_id
	             AND cm.user_id = ?
	       )`,
		authID, authID,
	)
	if err != nil {
		return nil, fmt.Errorf("mark messages delivered: %w", err)
	}

	// find conversations where this user is a member
	rows, err := db.c.Query(
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
	     JOIN conversation_members cm
	         ON cm.conversation_id = c.id
	     LEFT JOIN messages m ON m.id = (
	         SELECT m2.id
	         FROM messages m2
	         WHERE m2.conversation_id = c.id
	         ORDER BY m2.created_at DESC, m2.id DESC
	         LIMIT 1
	     )
	     WHERE cm.user_id = ?
	     ORDER BY
	         (m.created_at IS NULL) ASC,
	         m.created_at DESC,
	         c.created_at DESC,
	         c.id DESC`,
		authID,
	)

	if err != nil {
		return nil, fmt.Errorf("get conversations: %w", err)
	}

	conversations := make([]ConversationSummary, 0)

	for rows.Next() {
		var conversation ConversationSummary
		// Group photo
		var groupMime sql.NullString
		var groupPhoto []byte

		err := rows.Scan(
			&conversation.ID,
			&conversation.Type,
			&conversation.Name,
			&groupMime,
			&groupPhoto,
			&conversation.LastMessageAt,
			&conversation.LastMessageType,
			&conversation.LastMessagePreview,
		)

		if err != nil {
			closeRows(rows)
			return nil, fmt.Errorf("read conversation: %w", err)
		}

		if conversation.Type == "group" &&
			groupMime.Valid && len(groupPhoto) > 0 {

			conversation.Photo = &Photo{
				MimeType: groupMime.String,
				Data:     groupPhoto,
			}
		}

		conversations = append(conversations, conversation)

	}

	if err := rows.Err(); err != nil {
		closeRows(rows)
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
				closeRows(memberRows)
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
			closeRows(memberRows)
			return nil, fmt.Errorf("iterate members: %w", err)
		}
		if err := memberRows.Close(); err != nil {
			return nil, fmt.Errorf("close members: %w", err)
		}
	}

	return conversations, nil
}
