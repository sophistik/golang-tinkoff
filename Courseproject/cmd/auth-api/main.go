package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"../../internal/background"
	"../../internal/postgres"
	"go.uber.org/zap"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Config struct {
	ListenAddr  string
	DB          postgres.Config
	Base64DBURL string
}

func parseFlags() Config {
	var cfg Config

	kingpin.Flag("listen-addr", "Listen address.").
		Envar("LISTEN_ADDR").Default("8000").
		StringVar(&cfg.ListenAddr)

	kingpin.Flag("db-url", "DB URL.").
		Envar("DB_URL").Default("postgres://postgres:postgres@localhost:5432/fintech?sslmode=disable&fallback_application_name=courseproject").
		StringVar(&cfg.DB.URL)
	kingpin.Flag("db-max-conn", "DB max connections.").
		Envar("DB_MAX_CONN").Default("2").
		IntVar(&cfg.DB.MaxConnections)
	kingpin.Flag("db-max-conn-lifetime", "DB max connection lifetime.").
		Envar("DB_MAX_CONN_LIFETIME").Default("5m").
		DurationVar(&cfg.DB.MaxConnLifetime)
	kingpin.Flag("base64-db-url", "DB URL encoded by base64.").
		Envar("BASE64_DB_URL").Default("").
		StringVar(&cfg.Base64DBURL)

	kingpin.Parse()

	if cfg.Base64DBURL != "" {
		dbURL, err := base64.StdEncoding.DecodeString(cfg.Base64DBURL)
		if err != nil {
			log.Fatalf("Can't parse base64 db url: %s", err)
		}

		cfg.DB.URL = strings.TrimSpace(string(dbURL))
	}

	return cfg
}

func main() {
	cfg := parseFlags()
	logger, err := zap.NewDevelopment()

	if err != nil {
		log.Fatal("Can't create zap logger: ", err)
	}

	defer logger.Sync() // nolint:errcheck

	db, err := postgres.New(logger, cfg.DB)
	if err != nil {
		logger.Sugar().Fatalf("Can't create db: %s", err)
	}

	defer handleCloser(logger, "db", db)

	userStorage, err := postgres.NewUserStorage(db)
	if err != nil {
		logger.Sugar().Fatalf("Can't create user storage: %s", err)
	}

	defer handleCloser(logger, "user_storage", userStorage)

	sessionStorage, err := postgres.NewSessionStorage(db)
	if err != nil {
		logger.Sugar().Fatalf("Can't create session storage: %s", err)
	}

	defer handleCloser(logger, "session_storage", sessionStorage)

	robotStorage, err := postgres.NewRobotStorage(db)
	if err != nil {
		logger.Sugar().Fatalf("Can't create robot storage: %s", err)
	}

	defer handleCloser(logger, "robot_storage", robotStorage)

	h, err := NewHandler(logger, userStorage, sessionStorage, robotStorage)
	if err != nil {
		logger.Sugar().Fatalf("Can't create server: %s", err)
	}

	r := h.NewRouter()
	addr := net.JoinHostPort("", cfg.ListenAddr)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	sigquit := make(chan os.Signal, 1)

	signal.Ignore(syscall.SIGHUP, syscall.SIGPIPE)
	signal.Notify(sigquit, syscall.SIGINT, syscall.SIGTERM)

	stopAppCh := make(chan struct{})

	go background.NewBackground(h.logger, h.robotStorage, h.robotsChan)

	go func() {
		s := <-sigquit
		fmt.Printf("captured signal: %v\n", s)
		fmt.Println("gracefully shutting down server")

		if err := srv.Shutdown(context.Background()); err != nil {
			log.Fatalf("could not shutdown server: %s", err)
		}

		fmt.Println("server stopped")
		stopAppCh <- struct{}{}
	}()

	logger.Sugar().Infof("Server started on %s", cfg.ListenAddr)
	fmt.Printf("starting server, listening on %s\n", cfg.ListenAddr)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logger.Sugar().Fatalf("Can't serve requests: %s", err)
	}

	<-stopAppCh
}

func handleCloser(logger *zap.Logger, resource string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		logger.Sugar().Errorf("Can't close %q: %s", resource, err)
	}
}
