package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
	_ "github.com/spf13/viper"
)

func Validate(w http.ResponseWriter, r *http.Request) {
	user, err := getRequestUser(r)
	if err != nil {
		gores.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	w.Header().Set("X-Heracles-User", user.Username)

	gores.NoContent(w)
}
