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

type commentMessageRequest struct {
	Emoticon string `json:"emoticon"`
}

func (rt *_router) commentMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := chi.URLParam(r, "messageId")

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var request commentMessageRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	emoticon := strings.TrimSpace(request.Emoticon)

	if emoticon == "" {
		http.Error(w, "Emoticon is required", http.StatusBadRequest)
		return
	}

	reaction, err := rt.db.CommentMessage(
		parts[1], messageID, emoticon,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, database.ErrReactionMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrReactionForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot add reaction")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(reaction); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode reaction")
	}
}

func (rt *_router) uncommentMessage(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := chi.URLParam(r, "messageId")

	err := rt.db.UncommentMessage(parts[1], messageID)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, database.ErrReactionMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrReactionForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot delete reaction")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
