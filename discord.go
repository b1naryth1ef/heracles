package heracles

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const (
	authURL      string = "https://discordapp.com/api/oauth2/authorize"
	tokenURL     string = "https://discordapp.com/api/oauth2/token"
	userEndpoint string = "https://discordapp.com/api/v7/users/@me"
)

var cachedConfig *oauth2.Config

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"string"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
}

func InitializeDiscordAuth() {
	cachedConfig = &oauth2.Config{
		ClientID:     viper.GetString("discord.client_id"),
		ClientSecret: viper.GetString("discord.client_secret"),
		RedirectURL:  viper.GetString("discord.redirect_uri"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		Scopes: []string{"identify"},
	}
}

func GetLoginDiscordRoute(w http.ResponseWriter, r *http.Request) {
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

	url := cachedConfig.AuthCodeURL(session.Values["state"].(string), oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GetLoginDiscordCallbackRoute(w http.ResponseWriter, r *http.Request) {
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

	token, err := cachedConfig.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		reportInternalError(w, err)
		return
	}

	req, err := http.NewRequest("GET", userEndpoint, nil)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	req.Header.Set("Authorization", token.Type()+" "+token.AccessToken)
	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.Do(req)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	var discordUser DiscordUser
	err = json.NewDecoder(res.Body).Decode(&discordUser)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	id, err := strconv.ParseInt(discordUser.ID, 10, 64)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	var user *db.User

	user, err = db.GetUserByDiscordId(id)
	if err == sql.ErrNoRows {
		if viper.GetBool("discord.create") {
			user, err = db.CreateUser(discordUser.Username, "", 0, &id)
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
		"discord": id,
	})
	if err != nil {
		reportInternalError(w, err)
		return
	}

	cookie := http.Cookie{
		Name:   "heracles-auth",
		Domain: viper.GetString("web.domain"),
		Value:  authSecretEncoded,
		Path:   "/",
		MaxAge: 60 * 60 * 24 * 14,
	}
	http.SetCookie(w, &cookie)

	redirectURL := session.Values["r"].(string)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
