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

	"github.com/Sheff1981/boostlab-control/internal/api"
	"github.com/Sheff1981/boostlab-control/internal/config"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	registry := api.NewRegistry()

	seedNodes, err := api.ParseSeedNodes(os.Getenv("BOOSTLAB_NODES_JSON"))
	if err != nil {
		log.Error("invalid node bootstrap configuration", "error", err)
		os.Exit(2)
	}
	for _, node := range seedNodes {
		registry.Upsert(node)
	}
	log.Info("bootstrap nodes loaded", "count", len(seedNodes))

	games, err := api.ParseGameCatalog(os.Getenv("BOOSTLAB_GAMES_JSON"))
	if err != nil {
		log.Error("invalid game catalog configuration", "error", err)
		os.Exit(2)
	}
	log.Info("game catalog loaded", "count", len(games))

	routeTargets, err := api.ParseGameRouteTargets(os.Getenv("BOOSTLAB_GAME_ROUTES_JSON"))
	if err != nil {
		log.Error("invalid game route target configuration", "error", err)
		os.Exit(2)
	}
	log.Info("game route targets loaded", "count", len(routeTargets))

	social, err := api.NewPersistentSocialHub(cfg.SocialDataFile)
	if err != nil {
		log.Error("failed to load social history", "error", err)
		os.Exit(2)
	}
	log.Info("social history loaded", "path", cfg.SocialDataFile)

	voiceIce, err := api.ParseVoiceIceProvider(
		os.Getenv("BOOSTLAB_TURN_URLS"),
		os.Getenv("BOOSTLAB_TURN_SECRET"),
		os.Getenv("BOOSTLAB_TURN_TTL_SECONDS"),
	)
	if err != nil {
		log.Error("invalid TURN configuration", "error", err)
		os.Exit(2)
	}
	log.Info("voice ICE configuration loaded", "servers", len(voiceIce.TurnURLs))

	authHub, err := api.NewPersistentDeviceAuthHub(
		cfg.DeviceAuthDataFile,
		cfg.EnrollmentCode,
	)
	if err != nil {
		log.Error("failed to load device auth registry", "error", err)
		os.Exit(2)
	}
	log.Info("device auth registry loaded", "path", cfg.DeviceAuthDataFile)

	apiServer := api.NewServerWithFullDependencies(
		registry,
		games,
		routeTargets,
		social,
		voiceIce,
	)
	apiServer.Auth = authHub
	apiServer.ProvisioningSecret = []byte(cfg.ProvisioningSecret)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiServer.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("control API listening", "addr", cfg.HTTPAddr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("control API stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
