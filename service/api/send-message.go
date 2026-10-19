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

	"github.com/go-chi/chi/v5"
)

type sendMessageRequest struct {
	Type             string          `json:"type"`
	Text             string          `json:"text,omitempty"`
	Photo            *database.Photo `json:"photo,omitempty"`
	ReplyToMessageID string          `json:"replyToMessageId,omitempty"`
}

func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID := chi.URLParam(r, "conversationId")

	// limit the HTTP body to 10 MiB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	var request sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	switch request.Type {
	case "text":
		if strings.TrimSpace(request.Text) == "" || request.Photo != nil {
			http.Error(w, "Invalid text message", http.StatusBadRequest)
			return
		}

	case "photo":
		if request.Photo == nil || request.Text != "" {
			http.Error(w, "Invalid photo message", http.StatusBadRequest)
			return
		}

		_, format, err := image.DecodeConfig(
			bytes.NewReader(request.Photo.Data),
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
			request.Photo.MimeType != expectedMime[format] {
			http.Error(w, "Unsupported image type", http.StatusBadRequest)
			return
		}

	default:
		http.Error(w, "Invalid message type", http.StatusBadRequest)
		return
	}

	message, err := rt.db.SendMessage(
		parts[1],
		conversationID,
		database.SendMessageInput{
			Type:             request.Type,
			Text:             request.Text,
			Photo:            request.Photo,
			ReplyToMessageID: request.ReplyToMessageID,
		},
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrConversationForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if errors.Is(err, database.ErrInvalidReply) {
		http.Error(w, "Invalid reply", http.StatusBadRequest)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot send message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(message); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode message")
	}
}
