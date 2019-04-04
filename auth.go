package heracles

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"html/template"
	"net/http"
	"net/url"

	"github.com/alioygur/gores"
	"github.com/spf13/viper"
)

var ErrNoUser = errors.New("No User")

func getCurrentUser(r *http.Request) *User {
	return r.Context().Value("authuser").(*User)
}

func getRequestUserViaCookie(r *http.Request) (*User, error) {
	authCookie, err := r.Cookie("heracles-auth")
	if err != nil {
		return nil, err
	}

	decodedAuthSecret, err := base64.RawURLEncoding.DecodeString(authCookie.Value)
	if err != nil {
		return nil, err
	}

	return GetUserByAuthSecret(decodedAuthSecret)
}

func getRequestUserViaBasicAuth(r *http.Request) (*User, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, ErrNoUser
	}

	user, err := GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	if user.CheckPassword(password) != nil {
		return nil, ErrNoUser
	}

	return user, nil
}

func getRequestUser(r *http.Request) (*User, error) {
	user, err := getRequestUserViaCookie(r)
	if err == nil {
		return user, nil
	}

	user, err = getRequestUserViaBasicAuth(r)
	if err == nil {
		return user, nil
	}

	return nil, ErrNoUser
}

func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := getRequestUser(r)
		if err != nil {
			gores.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "authuser", user)))
	})
}

func GetLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	t := template.Must(template.New("login.html").ParseFiles("login.html"))
	t.Execute(w, r.Form.Get("r"))
	// http.ServeFile(w, r, "login.html")
}

func PostLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		gores.Error(w, http.StatusBadRequest, "Bad Form Data")
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	user, err := GetUserByUsername(username)
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

	authSecret := user.GetAuthSecret()
	authSecretEncoded := base64.RawURLEncoding.EncodeToString(authSecret)

	realmURL, err := url.Parse("https://" + r.Form.Get("r"))
	if err != nil {
		gores.Error(w, http.StatusBadRequest, "Bad realm")
		return
	}

	cookie := http.Cookie{
		Name:   "heracles-auth",
		Domain: viper.GetString("cookie_domain"),
		Value:  authSecretEncoded,
		Path:   "",
	}

	http.SetCookie(w, &cookie)
	// TODO: set max age

	http.Redirect(w, r, realmURL.String(), http.StatusFound)
}

func GetLogout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:   "heracles-auth",
		Domain: viper.GetString("cookie_domain"),
		MaxAge: -1,
	}

	http.SetCookie(w, &cookie)
	gores.NoContent(w)
}

func GetIdentity(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	gores.JSON(w, http.StatusOK, map[string]interface{}{
		"username": user.Username,
	})
}
