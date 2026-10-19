// New File

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

type createConversationRequest struct {
	RecipientID string `json:"recipientId"`
}

func (rt *_router) createConversation(w http.ResponseWriter, r *http.Request) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request createConversationRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(request.RecipientID) == "" {
		http.Error(w, "Recipient ID is required", http.StatusBadRequest)
		return
	}

	conversation, err := rt.db.CreateConversation(
		parts[1],
		request.RecipientID,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrRecipientNotFound) {
		http.Error(w, "Recipient not found", http.StatusNotFound)
		return
	}

	if errors.Is(err, database.ErrInvalidRecipient) {
		http.Error(w, "Cannot create a chat with yourself", http.StatusBadRequest)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(conversation); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode conversation")
	}
}
