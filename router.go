package heracles

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alioygur/gores"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const timeout = 15 * time.Second

func RequestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf("[%vms] %v %v", int(time.Now().Sub(start).Seconds()*1000), r.Method, r.URL)

	})
}

func reportInternalError(w http.ResponseWriter, err error) {
	gores.Error(w, http.StatusInternalServerError, fmt.Sprintf("Internal Error: %v", err))
}

func NewRouter() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(timeout))
	// router.Use(RequestLogMiddleware)

	router.Get("/login", GetLogin)
	router.Post("/login", PostLogin)
	router.Get("/logout", GetLogout)
	router.Handle("/validate", http.HandlerFunc(Validate))

	authRouter := router.With(RequireAuthMiddleware)

	authRouter.Get("/identity", GetIdentity)

	authRouter.Route("/users", func(r chi.Router) {
		// r.Get("/", ListUsers)
		r.Post("/", PostUsers)
	})

	// router.With(RequireUserMiddleware).Route("/users/{user_id}", func(r chi.Router) {
	// 	r.Get("/", GetUser)
	// 	r.Patch("/", PatchUser)
	// 	r.Delete("/", DeleteUser)
	// })

	return router
}
