package heracles

import (
	"database/sql"
	"net/http"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
)

func GetTokensRoute(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	tokens, err := db.GetUserTokensByUserId(user.Id)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
	})
}

type CreateTokenPayload struct {
	Name         string `json:"name" schema:"name"`
	UserId       *int64 `json:"user_id" schema:"user_id"`
	CanAccessAPI *bool  `json:"can_access_api" schema:"can_access_api"`
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
		user, err = db.GetUserById(*payload.UserId)
		if err == sql.ErrNoRows {
			gores.Error(w, http.StatusBadRequest, "Unknown User")
			return
		} else if err != nil {
			reportInternalError(w, err)
			return
		}
	}

	var flags db.Bits
	if payload.CanAccessAPI == nil || *payload.CanAccessAPI {
		flags = flags.Set(db.USER_TOKEN_FLAG_API)
	}

	token, err := db.CreateUserToken(user.Id, payload.Name, flags)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, token)
}

func DeleteTokenRoute(w http.ResponseWriter, r *http.Request) {
	userToken := getCurrentUserToken(r)
	err := userToken.Delete()
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.NoContent(w)
}

type PatchTokenPayload struct {
	Name       *string `json:"name"`
	ResetToken bool    `json:"reset_token"`
}

func PatchTokenRoute(w http.ResponseWriter, r *http.Request) {
	var payload PatchTokenPayload
	if !readRequestData(w, r, &payload) {
		return
	}

	userToken := getCurrentUserToken(r)
	if payload.ResetToken {
		newToken, err := db.GenerateUserTokenContents()
		if err != nil {
			reportInternalError(w, err)
			return
		}

		userToken.Token = newToken
	}

	if payload.Name != nil {
		userToken.Name = *payload.Name
	}

	err := userToken.Save()
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, userToken)
}
