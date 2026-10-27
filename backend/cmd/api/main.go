package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/org/metaesquemas/internal/adapters/store"
	"github.com/org/metaesquemas/internal/application"
	"github.com/org/metaesquemas/internal/config"
	"github.com/org/metaesquemas/internal/transport/httpapi"
	"github.com/org/metaesquemas/internal/transport/middleware"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	catalog, err := store.New(cfg.StoreDir, cfg.ReloadEvery, log)
	if err != nil {
		log.Error("store", "err", err)
		os.Exit(1)
	}
	svc := &application.Service{Catalog: catalog, Ingest: store.NewQueue(cfg.InboxDir, cfg.JobsDir), MaxUpload: cfg.MaxUploadBytes}

	geo, err := middleware.NewGeoBlocker(cfg.Geo, log)
	if err != nil {
		log.Error("geoip", "err", err)
		os.Exit(1)
	}
	defer geo.Close()

	srv := &http.Server{
		Addr: cfg.Addr, Handler: httpapi.NewRouter(cfg, svc, geo, log),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
