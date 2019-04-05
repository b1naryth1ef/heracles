package heracles

import (
	"database/sql"
	"net/http"

	"github.com/alioygur/gores"
)

func GetTokensRoute(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	tokens, err := GetUserTokensByUserId(user.Id)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
	})
}

type CreateTokenPayload struct {
	Name   string `json:"name"`
	UserId *int64 `json:"user_id"`
}

func PostTokensRoute(w http.ResponseWriter, r *http.Request) {
	var payload CreateTokenPayload
	if !readRequestData(w, r, &payload) {
		return
	}

	user := getCurrentUser(r)

	if payload.UserId != nil {
		if !user.IsAdmin() {
			gores.Error(w, http.StatusForbidden, "Cannot create tokens for another user")
			return
		}

		var err error
		user, err = GetUserById(*payload.UserId)
		if err == sql.ErrNoRows {
			gores.Error(w, http.StatusBadRequest, "Unknown User")
			return
		} else if err != nil {
			reportInternalError(w, err)
			return
		}
	}

	token, err := CreateUserToken(user.Id, payload.Name)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, token)
}
