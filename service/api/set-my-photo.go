// New File

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var photo database.Photo

	if err := json.NewDecoder(r.Body).Decode(&photo); err != nil {
		http.Error(w, "Invalid JSON or image data", http.StatusBadRequest)
		return
	}

	if len(photo.Data) == 0 {
		http.Error(w, "Photo data is required", http.StatusBadRequest)
		return
	}

	_, format, err := image.DecodeConfig(bytes.NewReader(photo.Data))
	if err != nil {
		http.Error(w, "Invalid image", http.StatusBadRequest)
		return
	}

	expectedMime := map[string]string{
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
	}

	if expectedMime[format] == "" || photo.MimeType != expectedMime[format] {
		http.Error(w, "Unsupported image type", http.StatusBadRequest)
		return
	}

	user, err := rt.db.SetMyPhoto(parts[1], photo)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot update photo")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode user response")
	}
}
