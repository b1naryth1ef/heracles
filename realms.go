package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
)

type CreateRealmPayload struct {
	Name string `json:"name" schema:"name"`
}

func GetRealmsRoute(w http.ResponseWriter, r *http.Request) {
	realms, err := db.GetRealms()
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, map[string]interface{}{
		"realms": realms,
	})
}

func PostRealmsRoute(w http.ResponseWriter, r *http.Request) {
	var payload CreateRealmPayload
	if !readRequestData(w, r, &payload) {
		return
	}

	if payload.Name == "" {
		gores.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	realm, err := db.CreateRealm(payload.Name)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, realm)
}

type CreateUserRealmGrantPayload struct {
	UserId int64   `json:"user_id" schema:"user_id"`
	Alias  *string `json:"alias" schema:"alias"`
}

func PostRealmsGrantsRoute(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserRealmGrantPayload
	if !readRequestData(w, r, &payload) {
		return
	}

	realm := getCurrentRealm(r)

	user, err := db.GetUserById(payload.UserId)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	realmGrant, err := db.CreateUserRealmGrant(user.Id, realm.Id, payload.Alias)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, realmGrant)
}
