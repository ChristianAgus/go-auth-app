package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth-app/internal/config"
	"auth-app/internal/database"
	"auth-app/internal/handler"
	"auth-app/internal/middleware"
	"auth-app/internal/repository"
	"auth-app/internal/service"
)

func main() {
	cfg := config.Load()

	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET wajib diisi di .env, minimal 32 karakter")
	}

	db, err := database.NewMySQL(cfg.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("terhubung ke MySQL")

	// wiring: repository -> service -> handler
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, time.Duration(cfg.JWTExpiryHours)*time.Hour)
	authHandler := handler.NewAuthHandler(authService)
	requireAuth := middleware.Auth(authService)
	noteRepo := repository.NewNoteRepository(db)
	noteService := service.NewNoteService(noteRepo)
	noteHandler := handler.NewNoteHandler(noteService)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "database down", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.Handle("GET /api/me", requireAuth(http.HandlerFunc(authHandler.Me)))
	mux.Handle("POST /api/notes", requireAuth(http.HandlerFunc(noteHandler.Create)))
	mux.Handle("GET /api/notes", requireAuth(http.HandlerFunc(noteHandler.List)))
	mux.Handle("GET /api/notes/{id}", requireAuth(http.HandlerFunc(noteHandler.Get)))
	mux.Handle("PUT /api/notes/{id}", requireAuth(http.HandlerFunc(noteHandler.Update)))
	mux.Handle("DELETE /api/notes/{id}", requireAuth(http.HandlerFunc(noteHandler.Delete)))

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("server jalan di http://localhost:%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("mematikan server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}