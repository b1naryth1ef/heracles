package heracles

import (
	"context"
	"database/sql"
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
	"github.com/go-chi/chi"
)

func getCurrentRealm(r *http.Request) *db.Realm {
	return r.Context().Value("realm").(*db.Realm)
}

func getCurrentUserToken(r *http.Request) *db.UserToken {
	return r.Context().Value("userToken").(*db.UserToken)
}

func getCurrentUser(r *http.Request) *db.User {
	return r.Context().Value("authuser").(*db.User)
}

func findRequestUserViaCookie(r *http.Request) (*db.User, error) {
	authCookie, err := r.Cookie("heracles-auth")
	if err != nil {
		return nil, err
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(authCookie.Value)
	if err != nil {
		return nil, err
	}

	return db.GetUserByAuthSecret(decodedAuthSecret)
}

func findRequestUserViaBasicAuth(r *http.Request, isAPI bool) (*db.User, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, ErrNoUser
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Check token first because its actually cheaper than a bcrypt check
	tokenUser, err := db.GetUserByToken(password, isAPI)
	if err == nil && tokenUser.Id == user.Id {
		return user, nil
	} else if user.CheckPassword(password) == nil {
		return user, nil
	}

	return nil, ErrNoUser
}

func findRequestUserViaAuthHeader(r *http.Request, isAPI bool) (*db.User, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, ErrNoUser
	}

	user, err := db.GetUserByToken(token, isAPI)
	if err == nil {
		return user, nil
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	// TODO: eventually this should be tokens
	return db.GetUserByAuthSecret(decodedAuthSecret)
}

func findRequestUser(r *http.Request, isAPI bool) (*db.User, error) {
	user, err := findRequestUserViaCookie(r)
	if err == nil {
		return user, nil
	}

	user, err = findRequestUserViaBasicAuth(r, isAPI)
	if err == nil {
		return user, nil
	}

	user, err = findRequestUserViaAuthHeader(r, isAPI)
	if err == nil {
		return user, nil
	}

	return nil, ErrNoUser
}

func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := findRequestUser(r, true)
		if err != nil {
			gores.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "authuser", user)))
	})
}

func RequireAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := getCurrentUser(r)
		if !user.IsAdmin() {
			gores.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
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

func RequireRealmMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realmIdRaw := chi.URLParam(r, "realmId")

		realmId, err := strconv.Atoi(realmIdRaw)
		if err != nil {
			gores.Error(w, http.StatusBadRequest, "Invalid realm ID")
			return
		}

		realm, err := db.GetRealmById(int64(realmId))
		if err == sql.ErrNoRows {
			gores.Error(w, http.StatusNotFound, "Not Found")
			return
		} else if err != nil {
			reportInternalError(w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "realm", realm)))
	})
}
