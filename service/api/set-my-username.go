package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

type usernameRequest struct {
	Name string `json:"name"`
}

func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request) {
	ctx := reqcontext.FromContext(r.Context())

	parts := strings.Fields(r.Header.Get("Authorization"))

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request usernameRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	request.Name = strings.TrimSpace(request.Name)

	length := utf8.RuneCountInString(request.Name)

	if length < 3 || length > 16 {
		http.Error(w, "Username must be between 3 and 16 characters",
			http.StatusBadRequest)
		return
	}

	user, err := rt.db.SetMyUserName(parts[1], request.Name)

	if errors.Is(err, database.ErrUnknownUser) {
		http.Error(w, "Unauthorized ", http.StatusUnauthorized)
		return
	}

	if errors.Is(err, database.ErrUsernameTaken) {
		http.Error(w, "Username already taken ", http.StatusConflict)
		return
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("cannot change username")
		http.Error(w, "Internal server error ", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode updated user ")
	}
}
