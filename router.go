package heracles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alioygur/gores"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const timeout = 15 * time.Second

func readRequestData(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(target)
		if err != nil {
			gores.Error(w, http.StatusBadRequest, fmt.Sprintf("Invalid Request Payload: %v", err))
			return false
		}

		return true
	}

	gores.Error(w, http.StatusBadRequest, fmt.Sprintf("Unsupported Content-Type: %v", contentType))
	return false
}

func reportInternalError(w http.ResponseWriter, err error) {
	gores.Error(w, http.StatusInternalServerError, fmt.Sprintf("Internal Error: %v", err))
}

func NewRouter() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(timeout))
	router.Use(middleware.Logger)

	router.Get("/login", GetLoginRoute)
	router.Post("/login", PostLoginRoute)
	router.Get("/logout", GetLogoutRoute)

	authRouter := router.With(RequireAuthMiddleware)
	authRouter.Get("/identity", GetIdentityRoute)

	authRouter.Handle("/validate", http.HandlerFunc(ValidateRoute))

	authRouter.Route("/tokens", func(r chi.Router) {
		r.Get("/", GetTokensRoute)
		r.Post("/", PostTokensRoute)
	})

	adminRouter := authRouter.With(RequireAdminMiddleware)

	adminRouter.Route("/users", func(r chi.Router) {
		r.Get("/", GetUsersRoute)
		r.Post("/", PostUsersRoute)
	})

	adminRouter.Route("/realms", func(r chi.Router) {
		r.Get("/", GetRealmsRoute)
		r.Post("/", PostRealmsRoute)
		r.Post("/grants", PostRealmsGrantsRoute)
	})

	// router.With(RequireUserMiddleware).Route("/users/{user_id}", func(r chi.Router) {
	// 	r.Get("/", GetUser)
	// 	r.Patch("/", PatchUser)
	// 	r.Delete("/", DeleteUser)
	// })

	return router
}
