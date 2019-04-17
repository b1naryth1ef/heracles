package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
)

type CreateUserPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

func PostUsersRoute(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserPayload

	if !readRequestData(w, r, &payload) {
		return
	}

	if payload.Username == "" || payload.Password == "" {
		gores.Error(w, http.StatusBadRequest, "username and password are required")
		return
	}

	var flags db.Bits
	if payload.Admin {
		flags = flags.Set(db.USER_FLAG_ADMIN)
	}

	user, err := db.CreateUser(payload.Username, payload.Password, flags)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, user)
}

func GetUsersRoute(w http.ResponseWriter, r *http.Request) {
	users, err := db.GetUsers()
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}
