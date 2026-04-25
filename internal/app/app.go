package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"example.com/taskservice/internal/handler"
	"example.com/taskservice/internal/repository"
	"example.com/taskservice/internal/service"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds all runtime configuration.
type Config struct {
	DatabaseURL string
	Addr        string
}

// App wires all components together.
type App struct {
	server *http.Server
	pool   *pgxpool.Pool
}

func New(cfg Config) (*App, error) {
	// ── Database ──────────────────────────────────────────────────────────────
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// ── Wire layers ───────────────────────────────────────────────────────────
	repo := repository.NewTaskRepository(pool)
	svc := service.NewTaskService(repo)
	h := handler.NewTaskHandler(svc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := mux.NewRouter()
	h.RegisterRoutes(r)

	// Health-check — useful for Docker / k8s probes.
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{server: srv, pool: pool}, nil
}

// Run starts the HTTP server and blocks until it returns.
func (a *App) Run() error {
	return a.server.ListenAndServe()
}

// Shutdown gracefully stops the server and closes the DB pool.
func (a *App) Shutdown(ctx context.Context) error {
	defer a.pool.Close()
	return a.server.Shutdown(ctx)
}
