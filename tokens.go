package heracles

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
	"github.com/go-chi/chi"
)

func getCurrentUserToken(r *http.Request) *db.UserToken {
	return r.Context().Value("userToken").(*db.UserToken)
}

func RequireUserTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenIdRaw := chi.URLParam(r, "tokenId")

		tokenId, err := strconv.Atoi(tokenIdRaw)
		if err != nil {
			gores.Error(w, http.StatusBadRequest, "Invalid token ID")
			return
		}

		userToken, err := db.GetUserTokenById(int64(tokenId))
		if err == sql.ErrNoRows {
			gores.Error(w, http.StatusNotFound, "Not Found")
			return
		} else if err != nil {
			reportInternalError(w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "userToken", userToken)))
	})
}

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
	Name   string `json:"name" schema:"name"`
	UserId *int64 `json:"user_id" schema:"user_id"`
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

	token, err := db.CreateUserToken(user.Id, payload.Name)
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
