package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

func (rt *_router) deleteMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	// Check Authorization header
	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get messageId from the URL
	messageID := chi.URLParam(r, "messageId")

	if messageID == "" {
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	// Delete message from SQL
	err := rt.db.DeleteMessage(parts[1], messageID)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	if errors.Is(err, database.ErrMessageForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot delete message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// successfully deleted: no JSON response body
	w.WriteHeader(http.StatusNoContent)
}
