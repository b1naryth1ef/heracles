package heracles

import (
	"net/http"

	"github.com/alioygur/gores"
	"github.com/b1naryth1ef/heracles/db"
)

func GetRecentAuditLogRoute(w http.ResponseWriter, r *http.Request) {
	entries, err := db.GetRecentAuditLogEntries(100)
	if err != nil {
		reportInternalError(w, err)
		return
	}

	gores.JSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
	})
}
