// Package main is the HTTP API entrypoint for the AI Video Assistant platform.
//
// @title           AI Video Assistant API
// @version         1.0
// @description     Meeting intelligence platform: submit a YouTube URL or file,
// @description     get transcript, summary, action items and a RAG chat interface.
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/raza-0003/ai-video-backend/internal/auth"
	"github.com/raza-0003/ai-video-backend/internal/config"
	"github.com/raza-0003/ai-video-backend/internal/db"
	"github.com/raza-0003/ai-video-backend/internal/handlers"
	"github.com/raza-0003/ai-video-backend/internal/queue"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	queueClient := queue.NewClient(cfg.RedisAddr)
	defer queueClient.Close()

	inspector := queue.NewInspector(cfg.RedisAddr)
	defer inspector.Close()

	authHandler := auth.NewHandler(pool, cfg.JWTSecret, cfg.JWTExpiryHrs)
	videoHandler := handlers.NewVideoHandler(pool, queueClient, inspector)
	dashboardHandler := handlers.NewDashboardHandler(pool)
	chatHandler := handlers.NewChatHandler(pool, cfg.PythonSvcURL)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/docs/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(cfg.JWTSecret))

			r.Route("/videos", func(r chi.Router) {
				r.Post("/", videoHandler.Submit)
				r.Get("/", videoHandler.History)
				r.Get("/{id}", videoHandler.Get)
				r.Post("/{id}/cancel", videoHandler.Cancel)
				r.Post("/{id}/chat", chatHandler.Chat)
				r.Get("/{id}/chat", chatHandler.History)
			})

			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/stats", dashboardHandler.Stats)
			})
		})
	})

	log.Printf("AI Video Assistant API listening on :%s (docs at /docs/index.html)", cfg.AppPort)
	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, r))
}
