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

	"github.com/go-chi/chi/v5"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

func (rt *_router) setGroupPhoto(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	// Check Authorization
	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := chi.URLParam(r, "groupId")

	if groupID == "" {
		http.Error(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Limit image upload size
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var photo database.Photo

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&photo); err != nil {
		http.Error(w, "Invalid JSON or image data", http.StatusBadRequest)
		return
	}

	if len(photo.Data) == 0 {
		http.Error(w, "Photo data is required", http.StatusBadRequest)
		return
	}

	// Validate actual image format
	_, format, err := image.DecodeConfig(
		bytes.NewReader(photo.Data),
	)

	if err != nil {
		http.Error(w, "Invalid image", http.StatusBadRequest)
		return
	}

	expectedMime := map[string]string{
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
	}

	if expectedMime[format] == "" ||
		photo.MimeType != expectedMime[format] {
		http.Error(w, "Unsupported image type", http.StatusBadRequest)
		return
	}

	// Save group photo
	group, err := rt.db.SetGroupPhoto(
		parts[1],
		groupID,
		photo,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if errors.Is(err, database.ErrGroupForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot update group photo")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(group); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode group")
	}
}
