package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

type forwardMessageRequest struct {
	ConversationID string `json:"conversationId"`
}

func (rt *_router) forwardMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sourceMessageID := chi.URLParam(r, "messageId")

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var request forwardMessageRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.ConversationID) == "" {
		http.Error(w, "Conversation ID is required", http.StatusBadRequest)
		return
	}

	message, err := rt.db.ForwardMessage(
		parts[1],
		sourceMessageID,
		request.ConversationID,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrMessageNotFound) ||
		errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if errors.Is(err, database.ErrConversationForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot forward message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(message); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode forwarded message")
	}
}
