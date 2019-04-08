package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
)

func PostUsersRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		gores.Error(w, http.StatusBadRequest, "Bad Form Data")
		return
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")
	admin := r.Form.Get("admin")

	if username == "" || password == "" {
		gores.Error(w, http.StatusBadRequest, "username and password are required")
		return
	}

	var flags db.Bits
	if admin == "1" {
		flags = flags.Set(db.USER_FLAG_ADMIN)
	}

	user, err := db.CreateUser(username, password, flags)
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
