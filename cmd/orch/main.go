package main

import (
  "flag"
  "fmt"
  "net/http"
  "os"
)

var (
  version = "dev"
  commit  = ""
  date    = ""
)

func main() {
  var showVersion bool
  var addr string

  flag.BoolVar(&showVersion, "version", false, "print version and exit")
  flag.StringVar(&addr, "addr", getEnv("ORCH_ADDR", ":8080"), "http listen address")
  flag.Parse()

  if showVersion {
    fmt.Printf("orch %s (commit=%s, date=%s)\n", version, commit, date)
    return
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


