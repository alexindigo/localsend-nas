// localsend-nas: send-only LocalSend node with an embedded web UI.
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

	"github.com/alexindigo/localsend-nas/internal/config"
	"github.com/alexindigo/localsend-nas/internal/discovery"
	"github.com/alexindigo/localsend-nas/internal/httpapi"
	"github.com/alexindigo/localsend-nas/internal/identity"
	"github.com/alexindigo/localsend-nas/internal/localsend"
	"github.com/alexindigo/localsend-nas/internal/lsserver"
	"github.com/alexindigo/localsend-nas/internal/settings"
	"github.com/alexindigo/localsend-nas/internal/shares"
	"github.com/alexindigo/localsend-nas/internal/transfer"
)

// version is stamped by release builds via -X main.version=...
var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(2)
	}

	id, err := identity.Load(cfg.DataDir)
	if err != nil {
		log.Error("identity", "error", err)
		os.Exit(1)
	}

	roots := map[string]string{}
	for _, sh := range cfg.Shares {
		roots[sh.Name] = sh.Path
	}
	store, err := shares.NewStore(roots)
	if err != nil {
		log.Error("shares", "error", err)
		os.Exit(1)
	}

	info := localsend.Info{
		Alias:       cfg.Alias,
		Version:     localsend.ProtocolVersion,
		DeviceModel: "localsend-nas",
		DeviceType:  "server",
		Fingerprint: id.Fingerprint,
		Port:        cfg.LSPort,
		Protocol:    "https",
		Download:    false,
	}

	settingsStore, err := settings.Load(cfg.DataDir)
	if err != nil {
		log.Error("settings", "error", err)
		os.Exit(1)
	}

	client := localsend.NewClient(id.Cert)
	disc := discovery.New(info, client, cfg.DataDir, log)
	lssrv := lsserver.New(cfg.LSPort, id, info, disc)
	tm := transfer.New(store, disc, client, info, log)

	webSrv := &http.Server{Addr: cfg.Listen, Handler: httpapi.New(cfg, store, disc, tm, settingsStore, info, version)}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	disc.Start(ctx)

	go func() {
		log.Info("localsend endpoint listening", "port", cfg.LSPort, "protocol", "https (send-only)")
		if err := lssrv.ListenAndServeTLS(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("localsend endpoint", "error", err)
			stop()
		}
	}()
	go func() {
		log.Info("web UI listening", "addr", cfg.Listen)
		if err := webSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("web UI", "error", err)
			stop()
		}
	}()

	names := store.Names()
	log.Info("localsend-nas started",
		"version", version,
		"alias", info.Alias,
		"fingerprint", id.Fingerprint,
		"shares", names,
	)

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	webSrv.Shutdown(shutdownCtx)
	lssrv.Shutdown(shutdownCtx)
}
