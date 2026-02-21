package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"microservices/pkg/authclient"
)

func NewRouter(mediaHandler *MediaHandler, authMiddleware *authclient.ChiMiddleware) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)

	// Routes
	r.Get("/health", mediaHandler.HealthCheck)

	r.Route("/media", func(r chi.Router) {
		// Apply auth middleware if available
		if authMiddleware != nil {
			r.Use(authMiddleware.RequireAuth)
		}

		r.Post("/upload-url", mediaHandler.GetUploadURL)
		r.Post("/confirm", mediaHandler.ConfirmUpload)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
