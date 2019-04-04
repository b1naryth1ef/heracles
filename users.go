package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
)

func PostUsers(w http.ResponseWriter, r *http.Request) {
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

	var flags Bits
	if admin == "1" {
		flags.Set(USER_FLAG_ADMIN)
	}

	user, err := CreateUser(username, password, flags)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, user)
}
