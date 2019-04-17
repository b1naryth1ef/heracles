package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
)

func GetIdentityRoute(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	gores.JSON(w, http.StatusOK, user)
}

type PatchIdentityPayload struct {
	Password string `json:"password" schema:"password"`
}

func PatchIdentityRoute(w http.ResponseWriter, r *http.Request) {
	var payload PatchIdentityPayload
	if !readRequestData(w, r, &payload) {
		return
	}

	user := getCurrentUser(r)

	err := user.UpdatePassword(payload.Password)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.NoContent(w)
}
