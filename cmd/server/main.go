package main

import (
	"log"
	"net/http"

	"mini-admin/internal/config"
	"mini-admin/internal/database"
	"mini-admin/internal/handlers"
	"mini-admin/internal/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}

	if err := database.SeedAdmin(db, cfg); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}

	authHandler := &handlers.AuthHandler{
		DB:              db,
		JWTSecret:       cfg.JWTSecret,
		AccessTokenTTL:  cfg.AccessTokenTTL,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
	}

	userHandler := &handlers.UserHandler{DB: db}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/refresh", authHandler.Refresh)

	authMw := middleware.AuthMiddleware(cfg.JWTSecret)
	activeMw := middleware.ActiveUserMiddleware(db)

	mux.Handle("/api/auth/me", authMw(activeMw(http.HandlerFunc(authHandler.Me))))
	mux.Handle("/api/auth/logout", authMw(http.HandlerFunc(authHandler.Logout)))

	mux.Handle("/api/users", authMw(activeMw(userHandler)))
	mux.Handle("/api/users/", authMw(activeMw(userHandler)))

	fs := http.FileServer(http.Dir("frontend"))
	mux.Handle("/", fs)

	handler := middleware.CORS(mux)

	log.Printf("Server starting on :%s", cfg.AppPort)
	log.Printf("Admin: %s / %s", cfg.SeedAdminEmail, cfg.SeedAdminPassword)
	if err := http.ListenAndServe(":"+cfg.AppPort, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
