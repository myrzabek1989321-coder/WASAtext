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

type setGroupNameRequest struct {
	Name string `json:"name"`
}

func (rt *_router) setGroupName(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	// Check Authorization header.
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

	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	var request setGroupNameRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	request.Name = strings.TrimSpace(request.Name)

	if request.Name == "" {
		http.Error(w, "Group name cannot be empty", http.StatusBadRequest)
		return
	}

	group, err := rt.db.SetGroupName(
		parts[1],
		groupID,
		request.Name,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrInvalidGroupName) {
		http.Error(w, "Invalid group name", http.StatusBadRequest)
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
		ctx.Logger.WithError(err).Error("cannot update group name")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(group); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode group")
	}
}
