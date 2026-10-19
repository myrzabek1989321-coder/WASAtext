// New file

package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

var ErrUsernameTaken = errors.New("Username already in use")

func (db *appdbimpl) SetMyUserName(authID, name string) (User, error) {
	var user User

	err := db.c.QueryRow(
		`UPDATE users
         SET name = ?
         WHERE id = ?
         RETURNING id, name`,
		name, authID,
	).Scan(&user.ID, &user.Name)

	if err == nil {
		return user, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnknownUser
	}

	var sqliteErr sqlite3.Error

	if errors.As(err, &sqliteErr) &&
		sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
		return User{}, ErrUsernameTaken
	}

	return User{}, fmt.Errorf("update username: %w", err)
}
