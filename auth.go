package heracles

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
)

var ErrNoUser = errors.New("No User")

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

	_, err = db.CreateAuditLogEntry("user.self_login", user, nil)
	if err != nil {
		reportInternalError(w, err)
		return
	}

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
	quiet := false

	quietArgs, ok := r.URL.Query()["quiet"]
	if ok && len(quietArgs) == 1 && quietArgs[0] == "1" {
		quiet = true
	}

	user, err := findRequestUser(r, false)
	if err != nil {
		if quiet {
			gores.NoContent(w)
		} else {
			gores.Error(w, http.StatusUnauthorized, "Unauthorized")
		}
		return
	}

	realm := r.Header.Get("X-Heracles-Realm")
	if realm == "" {
		if quiet {
			gores.NoContent(w)
		} else {
			log.Printf("[Validate] invalid realm provided: %v", realm)
			gores.Error(w, http.StatusBadRequest, "Invalid Realm")
		}
		return
	}

	realmGrant, err := db.GetUserRealmGrantByRealmName(user.Id, realm)
	if err != nil {
		if quiet {
			gores.NoContent(w)
		} else {
			log.Printf("[Validate] failed to find realm grant for user %v and realm %v: %v", user.Username, realm, err)
			gores.Error(w, http.StatusUnauthorized, "Unauthorized")
		}
		return
	}

	if realmGrant.Alias != nil {
		w.Header().Set("X-Heracles-User", *realmGrant.Alias)
	} else {
		w.Header().Set("X-Heracles-User", user.Username)
	}

	if user.DiscordId != nil {
		w.Header().Set("X-Heracles-DiscordID", fmt.Sprintf("%v", *user.DiscordId))
	}

	gores.NoContent(w)
}
