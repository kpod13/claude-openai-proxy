package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/timur/claude-code-openai-server/internal/config"
	"github.com/timur/claude-code-openai-server/internal/proxy"
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: search standard locations)")

	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Println("Discovering Claude models...")

	reg := proxy.Discover(cfg.Aliases)
	if reg.Len() == 0 {
		log.Fatal("no Claude models discovered — ensure the claude CLI is on PATH and authenticated")
	}

	log.Printf("Discovered %d model(s)", reg.Len())

	h := &proxy.Handler{Registry: reg}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprint(w, "ok")
		if err != nil {
			http.Error(w, "write error", http.StatusInternalServerError)

			return
		}
	})
	mux.HandleFunc("/v1/models", h.Models)
	mux.HandleFunc("/v1/chat/completions", h.ChatCompletions)

	srv := &http.Server{
		Addr:         cfg.Listen,
		Handler:      mux,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	log.Printf("Starting server on %s", cfg.Listen)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
