package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"warehouse-routing/internal/routing"
	"warehouse-routing/internal/store"
)

type candidateStore interface {
	Candidates(context.Context, string, int) ([]routing.Candidate, error)
	Reserve(context.Context, string, string, int) error
	Ping(context.Context) error
}

type app struct {
	store candidateStore
}

func main() {
	port := env("PORT", "8080")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := store.Open(ctx, dsn)
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	a := app{store: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", a.ready)
	mux.HandleFunc("POST /route", a.route)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}
	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a app) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a app) route(w http.ResponseWriter, r *http.Request) {
	var req routing.Request
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := routing.Validate(req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 150*time.Millisecond)
	defer cancel()

	candidates, err := a.store.Candidates(ctx, req.SKU, req.Quantity)
	if err != nil {
		slog.Error("load candidates", "err", err)
		writeError(w, http.StatusInternalServerError, "candidate_lookup_failed")
		return
	}

	decision, err := routing.Select(req, candidates)
	if errors.Is(err, routing.ErrNoCandidate) {
		writeError(w, http.StatusNotFound, "no_warehouse_available")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := a.store.Reserve(ctx, decision.WarehouseID, req.SKU, req.Quantity); errors.Is(err, store.ErrStockChanged) {
		writeError(w, http.StatusConflict, "stock_changed")
		return
	} else if err != nil {
		slog.Error("reserve stock", "err", err)
		writeError(w, http.StatusInternalServerError, "reservation_failed")
		return
	}

	writeJSON(w, http.StatusOK, decision)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
