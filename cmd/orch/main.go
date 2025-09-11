package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"gorm.io/gorm/logger"

	gormstore "github.com/wilhg/orch/pkg/store/gormstore"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	var showVersion bool
	var addr string
	var dsn string

	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.StringVar(&addr, "addr", getEnv("ORCH_ADDR", ":8080"), "http listen address")
	flag.StringVar(&dsn, "db-dsn", getEnv("ORCH_DB_DSN", "postgres://postgres:postgres@localhost:5432/orch?sslmode=disable"), "Postgres DSN for persistence")
	flag.Parse()

	if showVersion {
		fmt.Printf("orch %s (commit=%s, date=%s)\n", version, commit, date)
		return
	}

	// Initialize persistence store (Postgres via GORM)
	_, err := gormstore.Open(dsn, gormstore.WithLogger(logger.Default.LogMode(logger.Info)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize store: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{Addr: addr, Handler: mux}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func getEnv(key string, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
