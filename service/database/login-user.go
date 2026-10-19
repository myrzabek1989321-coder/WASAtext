// New file
package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// LoginUser returns an existing users ID/creates new User.
func (db *appdbimpl) LoginUser(name string) (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate user ID: %w", err)
	}

	newID := hex.EncodeToString(randomBytes)

	_, err := db.c.Exec(
		`INSERT INTO users (id, name)
         VALUES (?, ?)
         ON CONFLICT(name) DO NOTHING`,
		newID, name,
	)
	if err != nil {
		return "", fmt.Errorf("insert user: %w", err)
	}

	var id string

	err = db.c.QueryRow(
		"SELECT id FROM users WHERE name = ?",
		name,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}

	return id, nil
}
