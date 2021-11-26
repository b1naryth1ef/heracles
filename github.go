package heracles

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const (
	githubAuthURL      string = "https://github.com/login/oauth/authorize"
	githubTokenURL     string = "https://github.com/login/oauth/access_token"
	githubUserEndpoint string = "https://api.github.com/user"
)

var githubCachedConfig *oauth2.Config

type GithubUser struct {
	ID               int64  `json:"id"`
	Login            string `json:"login"`
	OrganizationsURL string `json:"organizations_url"`
}

func InitializeGithubAuth() {
	githubCachedConfig = &oauth2.Config{
		ClientID:     viper.GetString("github.client_id"),
		ClientSecret: viper.GetString("github.client_secret"),
		RedirectURL:  viper.GetString("github.redirect_uri"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  githubAuthURL,
			TokenURL: githubTokenURL,
		},
		Scopes: []string{"user", "read:org"},
	}
}

func GetLoginGithubRoute(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	if session == nil {
		return
	}

	err := r.ParseForm()
	if err != nil {
		gores.Error(w, http.StatusBadRequest, "Bad Form Data")
		return
	}

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

	session.Values["r"] = redirectURL.String()
	session.Values["state"] = randSeq(32)
	session.Save(r, w)

	url := githubCachedConfig.AuthCodeURL(session.Values["state"].(string), oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GetLoginGithubCallbackRoute(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	if session == nil {
		return
	}

	state := r.FormValue("state")
	if state != session.Values["state"] {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	errorMessage := r.FormValue("error")
	if errorMessage != "" {
		gores.Error(w, http.StatusBadRequest, fmt.Sprintf("Error: %v", errorMessage))
		return
	}

	token, err := githubCachedConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		reportInternalError(w, err)
		return
	}

	req, err := http.NewRequest("GET", githubUserEndpoint, nil)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	req.Header.Set("Authorization", "token "+token.AccessToken)
	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.Do(req)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	var githubUser GithubUser
	err = json.NewDecoder(res.Body).Decode(&githubUser)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	var user *db.User

	user, err = db.GetUserByGithubId(githubUser.ID)
	if err == sql.ErrNoRows {
		if viper.GetBool("github.create") {
			user, err = db.CreateUser(githubUser.Login, "", 0, nil, &githubUser.ID)
			if err != nil {
				reportInternalError(w, err)
				return
			}
		} else {
			reportInternalError(w, err)
			return
		}
	} else if err != nil {
		reportInternalError(w, err)
		return
	}

	authSecret := user.GetAuthSecret()
	authSecretEncoded := base64.RawURLEncoding.EncodeToString(authSecret)

	_, err = db.CreateAuditLogEntry("user.self_login", user, map[string]interface{}{
		"github": githubUser.ID,
	})
	if err != nil {
		reportInternalError(w, err)
		return
	}

	cookie := http.Cookie{
		Name:   "heracles-auth",
		Domain: "buildyboi.ci",
		Value:  authSecretEncoded,
		Path:   "/",
		MaxAge: 60 * 60 * 24 * 14,
	}
	http.SetCookie(w, &cookie)

	redirectURL := session.Values["r"].(string)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
