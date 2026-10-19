package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

type createGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

func (rt *_router) createGroup(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := reqcontext.FromContext(r.Context())

	// 1. Check Authorization
	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Read the request body
	r.Body = http.MaxBytesReader(w, r.Body, 65536)

	var request createGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 3. Validate the group name and member ids
	request.Name = strings.TrimSpace(request.Name)

	if request.Name == "" || request.MemberIDs == nil {
		http.Error(w, "Name and memberIds are required", http.StatusBadRequest)
		return
	}

	seen := make(map[string]bool)

	for _, id := range request.MemberIDs {
		if strings.TrimSpace(id) == "" || seen[id] {
			http.Error(w, "Invalid or duplicate member ID", http.StatusBadRequest)
			return
		}
		seen[id] = true
	}

	// 4. Create the group in SQLite
	group, err := rt.db.CreateGroup(
		parts[1],
		request.Name,
		request.MemberIDs,
	)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrInvalidGroupName) {
		http.Error(w, "Invalid group name", http.StatusBadRequest)
		return
	}

	if errors.Is(err, database.ErrGroupMemberNotFound) {
		http.Error(w, "Group member not found", http.StatusNotFound)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create group")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 5. Return the new group
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(group); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode group")
	}
}
