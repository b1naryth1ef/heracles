package heracles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alioygur/gores"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/schema"
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
	} else if contentType == "application/x-www-form-urlencoded" {
		err := r.ParseForm()
		if err != nil {
			gores.Error(w, http.StatusBadRequest, fmt.Sprintf("Invalid Request Form: %v", err))
			return false
		}

		err = schema.NewDecoder().Decode(target, r.PostForm)
		if err != nil {
			gores.Error(w, http.StatusBadRequest, fmt.Sprintf("Invalid Form Data: %v", err))
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

	authRouter := router.With(RequireAuthMiddleware)

	// Static/User-Friendly Routes
	router.Get("/login", GetLoginRoute)
	router.Post("/login", PostLoginRoute)
	router.Get("/login/discord", GetLoginDiscordRoute)
	router.Get("/login/discord/callback", GetLoginDiscordCallbackRoute)
	router.Handle("/logout", http.HandlerFunc(GetLogoutRoute))
	authRouter.Get("/", GetIndexRoute)

	// Validate route used for linking up nginx auth_request
	router.Handle("/api/validate", http.HandlerFunc(ValidateRoute))

	authRouter.Route("/api", func(apiRouter chi.Router) {
		// Returns information about the current users identity
		apiRouter.Get("/identity", GetIdentityRoute)

		// Updates the users identity
		apiRouter.Patch("/identity", PatchIdentityRoute)

		// Tokens can be managed by users and give third party services / clients
		//  access on behalf of a registered user.
		apiRouter.Route("/tokens", func(r chi.Router) {
			r.Get("/", GetTokensRoute)
			r.Post("/", PostTokensRoute)

			r.With(RequireUserTokenMiddleware).Route("/{tokenId}", func(r chi.Router) {
				r.Delete("/", DeleteTokenRoute)
				r.Patch("/", PatchTokenRoute)
			})
		})

		adminRouter := apiRouter.With(RequireAdminMiddleware)

		adminRouter.Route("/users", func(r chi.Router) {
			r.Get("/", GetUsersRoute)
			r.Post("/", PostUsersRoute)
		})

		adminRouter.Route("/realms", func(r chi.Router) {
			r.Get("/", GetRealmsRoute)
			r.Post("/", PostRealmsRoute)

			r.With(RequireRealmMiddleware).Route("/{realmId}", func(r chi.Router) {
				r.Post("/grants", PostRealmsGrantsRoute)
			})
		})

		adminRouter.Route("/log", func(r chi.Router) {
			r.Get("/recent", GetRecentAuditLogRoute)
		})
	})

	// router.With(RequireUserMiddleware).Route("/users/{user_id}", func(r chi.Router) {
	// 	r.Get("/", GetUser)
	// 	r.Patch("/", PatchUser)
	// 	r.Delete("/", DeleteUser)
	// })

	return router
}
