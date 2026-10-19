// New file

package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrUnknownUser = errors.New("unknown user")

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Photo *Photo `json:"photo,omitempty"`
}

func (db *appdbimpl) GetUsers(authID, search string) ([]User, error) {
	// Check that requesting user exists
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

	// Find users matching the provided search text.
	rows, err := db.c.Query(
		`SELECT id, name, photo_mime_type, photo_data
 		FROM users
 		WHERE instr(lower(name), lower(?)) > 0
 		ORDER BY name COLLATE NOCASE`,
		search,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var user User
		var mimeType sql.NullString
		var photoData []byte

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&mimeType,
			&photoData,
		); err != nil {
			return nil, fmt.Errorf("read user: %w", err)
		}

		if mimeType.Valid && len(photoData) > 0 {
			user.Photo = &Photo{
				MimeType: mimeType.String,
				Data:     photoData,
			}
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}
