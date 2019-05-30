package heracles

import (
	"github.com/b1naryth1ef/heracles/db"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func handleRadiusRequest(w radius.ResponseWriter, r *radius.Request) {
	username := rfc2865.UserName_GetString(r.Packet)
	password := rfc2865.UserPassword_GetString(r.Packet)

	user, err := db.GetUserByUsername(username)
	if err != nil {
		w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	if user.CheckPassword(password) != nil {
		w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	w.Write(r.Response(radius.CodeAccessAccept))
}
