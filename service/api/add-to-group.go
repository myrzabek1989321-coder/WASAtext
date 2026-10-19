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

type addToGroupRequest struct {
	MemberIDs []string `json:"memberIds"`
}

func (rt *_router) addToGroup(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := chi.URLParam(r, "groupId")

	r.Body = http.MaxBytesReader(w, r.Body, 65536)

	var request addToGroupRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(request.MemberIDs) == 0 {
		http.Error(w, "memberIds cannot be empty", http.StatusBadRequest)
		return
	}

	seen := make(map[string]bool)

	for _, userID := range request.MemberIDs {
		if strings.TrimSpace(userID) == "" || seen[userID] {
			http.Error(w, "Invalid or duplicate member ID", http.StatusBadRequest)
			return
		}
		seen[userID] = true
	}

	members, err := rt.db.AddToGroup(
		parts[1],
		groupID,
		request.MemberIDs,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrAddGroupNotFound) ||
		errors.Is(err, database.ErrAddGroupUserNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if errors.Is(err, database.ErrAddGroupForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot add group members")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(members); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode group members")
	}
}
