package heracles

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
)

var ErrNoUser = errors.New("No User")

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

func findRequestUserViaBasicAuth(r *http.Request) (*db.User, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, ErrNoUser
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Check token first because its actually cheaper than a bcrypt check
	tokenUser, err := db.GetUserByToken(password)
	if err == nil && tokenUser.Id == user.Id {
		return user, nil
	} else if user.CheckPassword(password) == nil {
		return user, nil
	}

	return nil, ErrNoUser
}

func findRequestUserViaAuthHeader(r *http.Request) (*db.User, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, ErrNoUser
	}

	user, err := db.GetUserByToken(token)
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

func findRequestUser(r *http.Request) (*db.User, error) {
	user, err := findRequestUserViaCookie(r)
	if err == nil {
		return user, nil
	}

	user, err = findRequestUserViaBasicAuth(r)
	if err == nil {
		return user, nil
	}

	user, err = findRequestUserViaAuthHeader(r)
	if err == nil {
		return user, nil
	}

	return nil, ErrNoUser
}

func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := findRequestUser(r)
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

func GetLoginRoute(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	t := template.Must(template.New("login.html").ParseFiles("static/login.html"))
	t.Execute(w, r.Form.Get("r"))
}

func GetIndexRoute(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("index.html").ParseFiles("static/index.html"))
	t.Execute(w, getCurrentUser(r))
}

func PostLoginRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		gores.Error(w, http.StatusBadRequest, "Bad Form Data")
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	user, err := db.GetUserByUsername(username)
	if err == sql.ErrNoRows {
		gores.Error(w, http.StatusBadRequest, "Unknown user")
		return
	} else if err != nil {
		reportInternalError(w, err)
		return
	}

	if user.CheckPassword(password) != nil {
		gores.Error(w, http.StatusBadRequest, "Bad password")
		return
	}

	// Create our authentication cookie
	authSecret := user.GetAuthSecret()
	authSecretEncoded := base64.RawURLEncoding.EncodeToString(authSecret)

	cookie := http.Cookie{
		Name:   "heracles-auth",
		Domain: viper.GetString("web.domain"),
		Value:  authSecretEncoded,
		Path:   "",
		MaxAge: 60 * 60 * 24 * 14,
	}
	http.SetCookie(w, &cookie)

	redirectURLRaw := r.Form.Get("r")
	if redirectURLRaw == "" {
		gores.NoContent(w)
		return
	}

	redirectURL, err := url.Parse(redirectURLRaw)
	if err != nil {
		gores.Error(w, http.StatusBadRequest, "Bad Redirect URL")
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func GetLogoutRoute(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:   "heracles-auth",
		Domain: viper.GetString("web.domain"),
		Value:  "",
		Path:   "",
		MaxAge: -1,
	}

	http.SetCookie(w, &cookie)

	if r.Method == "GET" {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	gores.NoContent(w)
}

func ValidateRoute(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	realm := r.Header.Get("X-Heracles-Realm")
	if realm == "" {
		log.Printf("[Validate] invalid realm provided: %v", realm)
		gores.Error(w, http.StatusBadRequest, "Invalid Realm")
		return
	}

	realmGrant, err := db.GetUserRealmGrantByRealmName(user.Id, realm)
	if err != nil {
		log.Printf("[Validate] failed to find realm grant for user %v and realm %v: %v", user.Username, realm, err)
		gores.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if realmGrant.Alias != nil {
		w.Header().Set("X-Heracles-User", *realmGrant.Alias)
	} else {
		w.Header().Set("X-Heracles-User", user.Username)
	}

	gores.NoContent(w)
}
