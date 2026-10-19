// New file

package database

import (
	"errors"
	"fmt"
)

var ErrUnknownUser = errors.New("unknown user")

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
		`SELECT id, name FROM users
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

		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			return nil, fmt.Errorf("read user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}
