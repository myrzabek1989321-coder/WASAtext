// New file

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

func (rt *_router) getUsers(w http.ResponseWriter, r *http.Request) {
	ctx := reqcontext.FromContext(r.Context())

	// Read the Authorization header.
	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authID := parts[1]
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	users, err := rt.db.GetUsers(authID, search)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot get users")
		http.Error(w, "Internal server error",
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(users); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode users")
	}
}
