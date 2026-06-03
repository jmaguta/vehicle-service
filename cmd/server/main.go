package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jmaguta/vehicle-service/internal/db"
	"github.com/jmaguta/vehicle-service/internal/handlers"
	mw "github.com/jmaguta/vehicle-service/internal/middleware"
	"github.com/jmaguta/vehicle-service/internal/vehicles"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	pool, err := db.Connect(context.Background())
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := vehicles.NewPostgresRepository(pool)
	ch := handlers.NewCustomerHandler(repo, log)
	vh := handlers.NewVehicleHandler(repo, log)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(mw.Logger(log))
	r.Use(chimw.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"error","reason":"db"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	idempotency := mw.Idempotency(pool)

	// Customers
	r.With(mw.RequireAuthOrServiceKey).Get("/customers", ch.List)
	r.With(mw.RequireAdminOrServiceKey, idempotency).Post("/customers", ch.Create)
	r.With(mw.RequireAuthOrServiceKey).Get("/customers/{id}", ch.Get)
	r.With(mw.RequireAdminOrServiceKey, idempotency).Patch("/customers/{id}", ch.Patch)

	// Vehicles
	r.With(mw.RequireAuthOrServiceKey).Get("/vehicles", vh.List)
	r.With(mw.RequireAdminOrServiceKey, idempotency).Post("/vehicles", vh.Create)
	r.With(mw.RequireAuthOrServiceKey).Get("/vehicles/{id}", vh.Get)
	r.With(mw.RequireAdminOrServiceKey, idempotency).Patch("/vehicles/{id}", vh.Patch)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
	}
}
