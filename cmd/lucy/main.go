// Command lucy is a personal uptime monitor.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"lucy/internal/api"
	"lucy/internal/checker"
	"lucy/internal/notify"
	"lucy/internal/scheduler"
	"lucy/internal/store"
	"lucy/web"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	command := run
	if len(os.Args) > 1 {
		if os.Args[1] != "reset-password" {
			fmt.Fprintln(os.Stderr, "kullanım: lucy [reset-password]")
			os.Exit(2)
		}
		command = resetPassword
	}
	if err := command(log); err != nil {
		log.Error("lucy stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := openStore()
	if err != nil {
		return err
	}
	defer db.Close()

	guard := checker.NewGuard(strings.Split(env("LUCY_PROBE_TARGETS", "1.1.1.1:443,8.8.8.8:443"), ","))
	check := checker.New()
	sched := &scheduler.Scheduler{Store: db, Checker: check, Guard: guard, Domains: checker.NewDomains(), Log: log}

	// The environment only seeds the first start; after that the settings screen owns these.
	if topic := os.Getenv("LUCY_NTFY_TOPIC"); topic != "" {
		seed := store.NtfySettings{
			Server: env("LUCY_NTFY_SERVER", store.DefaultNtfyServer),
			Topic:  topic,
			Token:  os.Getenv("LUCY_NTFY_TOKEN"),
		}
		if err := db.SeedNtfy(ctx, seed); err != nil {
			return err
		}
	}
	live := notify.Live{Store: db}
	sched.Composer = &notify.Composer{Language: env("LUCY_LANG", "en"), PublicURL: os.Getenv("LUCY_PUBLIC_URL")}
	sched.Enabled = live.Configured
	sched.Link = live.PublicURL
	go notify.Worker{Store: db, Notifier: live, Log: log}.Run(ctx)

	if err := sched.Start(ctx); err != nil {
		return err
	}
	go prune(ctx, db, log)

	server := &api.Server{Store: db, Scheduler: sched, Checker: check, Guard: guard, Web: web.Dist(), Log: log}
	// Loopback by default: there is no login yet, so exposing the API must be a choice.
	httpServer := &http.Server{
		Addr:              env("LUCY_ADDR", "127.0.0.1:8080"),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdown)
	}()
	log.Info("listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func openStore() (*store.Store, error) {
	dbPath := env("LUCY_DB", "data/lucy.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, err
	}
	return store.Open(dbPath)
}

func prune(ctx context.Context, db *store.Store, log *slog.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		if err := db.Prune(ctx, time.Now()); err != nil && ctx.Err() == nil {
			log.Error("prune", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
