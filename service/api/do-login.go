// New file

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
)

type loginRequest struct {
	Name string `json:"name"`
}

type loginResponse struct {
	Identifier string `json:"identifier"`
}

func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request) {
	ctx := reqcontext.FromContext(r.Context())

	var request loginRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	request.Name = strings.TrimSpace(request.Name)

	nameLength := utf8.RuneCountInString(request.Name)
	if nameLength < 3 || nameLength > 16 {
		http.Error(w, "Username must be between 3 and 16 characters",
			http.StatusBadRequest)
		return
	}

	id, err := rt.db.LoginUser(request.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot login user")
		http.Error(w, "Internal server error",
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(loginResponse{
		Identifier: id,
	}); err != nil {
		ctx.Logger.WithError(err).Error("cannot encode login response")
	}
}
