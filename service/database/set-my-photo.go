// New File
package database

import (
	"database/sql"
	"errors"
	"fmt"
)

type Photo struct {
	MimeType string `json:"mimeType"`
	Data     []byte `json:"data"`
}

func (db *appdbimpl) SetMyPhoto(authID string, photo Photo) (User, error) {
	var user User
	var mimeType string
	var data []byte

	err := db.c.QueryRow(
		`UPDATE users
         SET photo_mime_type = ?, photo_data = ?
         WHERE id = ?
         RETURNING id, name, photo_mime_type, photo_data`,
		photo.MimeType, photo.Data, authID,
	).Scan(&user.ID, &user.Name, &mimeType, &data)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnknownUser
	}

	if err != nil {
		return User{}, fmt.Errorf("update user photo: %w", err)
	}

	user.Photo = &Photo{
		MimeType: mimeType,
		Data:     data,
	}

	return user, nil
}
