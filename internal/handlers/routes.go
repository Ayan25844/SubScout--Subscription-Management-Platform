package handlers

import (
	"net/http"

	"github.com/Ayan25844/subscout/internal/database"
	"github.com/Ayan25844/subscout/internal/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func RegisterRoutes(db *database.Service) http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)

	r.Get("/swagger/*", httpSwagger.Handler())

	authHandler := &AuthHandler{DB: db}
	subHandler := &SubscriptionHandler{DB: db}

	r.Route("/api/v1", func(r chi.Router) {

		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware)
			r.Post("/categories", middleware.RoleBarrier("admin", subHandler.CreateCategory))
			r.Put("/categories/{id}", middleware.RoleBarrier("admin", subHandler.UpdateCategory))
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware)
			r.Get("/categories", subHandler.GetCategories)
			r.Get("/currencies", authHandler.GetCurrencies)
			r.Put("/users/me", authHandler.UpdateProfile)
			r.Put("/users/me/password", authHandler.UpdatePassword)
			r.Delete("/users/me/delete", authHandler.DeleteAccount)

			r.Route("/subscriptions", func(r chi.Router) {
				r.Post("/", subHandler.CreateSubscription)
				r.Get("/", subHandler.GetSubscriptions)
				r.Get("/{categoryID}", subHandler.GetSubscriptionsCategoryID)
				r.Put("/{id}", subHandler.UpdateSubscription)
				r.Put("/status/{id}", subHandler.UpdateSubscriptionStatus)
				r.Delete("/{id}", subHandler.DeleteSubscription)
			})
		})
	})
	return r
}
